package plugin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/plugin/proto"
	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/chat"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
)

type Manager struct {
	srv *server.Server
	log *slog.Logger

	ctx    context.Context
	cancel context.CancelFunc

	mu       sync.RWMutex
	plugins  map[string]*pluginProcess
	players  map[uuid.UUID]*player.Player
	commands map[string]commandBinding
}

type commandBinding struct {
	pluginID   string
	command    string
	descriptor *proto.CommandSpec
}

const eventResponseTimeout = 250 * time.Millisecond

func NewManager(srv *server.Server, log *slog.Logger) *Manager {
	if log == nil {
		log = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		srv:      srv,
		log:      log.With("component", "plugin-manager"),
		ctx:      ctx,
		cancel:   cancel,
		plugins:  make(map[string]*pluginProcess),
		players:  make(map[uuid.UUID]*player.Player),
		commands: make(map[string]commandBinding),
	}
}

func (m *Manager) Start(configPath string) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			m.log.Info("no plugin configuration found", "path", configPath)
			return nil
		}
		return err
	}
	for _, pc := range cfg.Plugins {
		if pc.ID == "" {
			pc.ID = pc.Name
		}
		if pc.ID == "" {
			pc.ID = fmt.Sprintf("plugin-%s", strings.ToLower(uuid.NewString()[:8]))
		}
		proc := newPluginProcess(m, pc)
		m.mu.Lock()
		m.plugins[pc.ID] = proc
		m.mu.Unlock()
		go proc.start(m.ctx)
	}
	return nil
}

func (m *Manager) Close() {
	m.cancel()
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, proc := range m.plugins {
		shutdown := &proto.HostToPlugin{PluginID: id, Shutdown: &proto.HostShutdown{Reason: "server shutting down"}}
		proc.queue(shutdown)
		proc.Stop()
	}
	m.plugins = make(map[string]*pluginProcess)
}

func (m *Manager) AttachWorld(w *world.World) {
	if w == nil {
		return
	}
	handler := &WorldHandler{mgr: m}
	w.Handle(handler)
}

func (m *Manager) AttachPlayer(p *player.Player) {
	if p == nil {
		return
	}
	handler := &PlayerHandler{mgr: m, Player: p}
	p.Handle(handler)
	m.mu.Lock()
	m.players[p.UUID()] = p
	m.mu.Unlock()
	m.emitPlayerJoin(p)
}

func (m *Manager) detachPlayer(p *player.Player) {
	m.mu.Lock()
	delete(m.players, p.UUID())
	m.mu.Unlock()
}

func (m *Manager) emitPlayerJoin(p *player.Player) {
	evt := &proto.EventEnvelope{
		EventID: generateEventID(),
		Type:    "PLAYER_JOIN",
		PlayerJoin: &proto.PlayerJoinEvent{
			PlayerUUID: p.UUID().String(),
			Name:       p.Name(),
		},
	}
	m.broadcastEvent(evt)
}

func (m *Manager) emitPlayerQuit(p *player.Player) {
	evt := &proto.EventEnvelope{
		EventID: generateEventID(),
		Type:    "PLAYER_QUIT",
		PlayerQuit: &proto.PlayerQuitEvent{
			PlayerUUID: p.UUID().String(),
			Name:       p.Name(),
		},
	}
	m.broadcastEvent(evt)
}

func (m *Manager) emitChat(ctx *player.Context, p *player.Player, msg *string) {
	if msg == nil {
		return
	}
	evt := &proto.EventEnvelope{
		EventID: generateEventID(),
		Type:    "CHAT",
		Chat: &proto.ChatEvent{
			PlayerUUID: p.UUID().String(),
			Name:       p.Name(),
			Message:    *msg,
		},
	}
	results := m.dispatchEvent(evt, true)
	var cancelled bool
	for _, res := range results {
		if res == nil {
			continue
		}
		if res.Cancel {
			cancelled = true
		}
		if res.Chat != nil {
			*msg = res.Chat.Message
		}
	}
	if cancelled && ctx != nil {
		ctx.Cancel()
	}
}

func (m *Manager) emitCommand(ctx *player.Context, p *player.Player, raw string) {
	evt := &proto.EventEnvelope{
		EventID: generateEventID(),
		Type:    "COMMAND",
		Command: &proto.CommandEvent{
			PlayerUUID: p.UUID().String(),
			Name:       p.Name(),
			Raw:        raw,
		},
	}
	results := m.dispatchEvent(evt, true)
	for _, res := range results {
		if res != nil && res.Cancel && ctx != nil {
			ctx.Cancel()
			break
		}
	}
}

func (m *Manager) emitBlockBreak(ctx *player.Context, p *player.Player, pos cube.Pos, drops *[]item.Stack, xp *int) {
	evt := &proto.EventEnvelope{
		EventID: generateEventID(),
		Type:    "BLOCK_BREAK",
		BlockBreak: &proto.BlockBreakEvent{
			PlayerUUID: p.UUID().String(),
			Name:       p.Name(),
			World:      fmt.Sprint(p.Tx().World().Dimension()),
			X:          int32(pos.X()),
			Y:          int32(pos.Y()),
			Z:          int32(pos.Z()),
		},
	}
	results := m.dispatchEvent(evt, true)
	var cancelled bool
	for _, res := range results {
		if res == nil {
			continue
		}
		if res.Cancel {
			cancelled = true
		}
		if res.BlockBreak != nil {
			if drops != nil {
				*drops = convertProtoDrops(res.BlockBreak.Drops)
			}
			if res.BlockBreak.XP != nil && xp != nil {
				*xp = int(*res.BlockBreak.XP)
			}
		}
	}
	if cancelled && ctx != nil {
		ctx.Cancel()
	}
}

func (m *Manager) broadcastEvent(envelope *proto.EventEnvelope) {
	_ = m.dispatchEvent(envelope, false)
}

func (m *Manager) dispatchEvent(envelope *proto.EventEnvelope, expectResult bool) []*proto.EventResult {
	if envelope == nil {
		return nil
	}
	eventType := strings.ToUpper(envelope.Type)
	m.mu.RLock()
	procs := make([]*pluginProcess, 0, len(m.plugins))
	for _, proc := range m.plugins {
		if !proc.HasSubscription(eventType) {
			continue
		}
		procs = append(procs, proc)
	}
	m.mu.RUnlock()

	if len(procs) == 0 {
		return nil
	}

	results := make([]*proto.EventResult, 0, len(procs))
	for _, proc := range procs {
		var waitCh chan *proto.EventResult
		if expectResult {
			waitCh = proc.expectEventResult(envelope.EventID)
		}
		msg := &proto.HostToPlugin{PluginID: proc.id, Event: envelope}
		proc.queue(msg)
		if !expectResult {
			continue
		}
		res, err := proc.waitEventResult(waitCh, eventResponseTimeout)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				proc.log.Warn("plugin did not respond to event", "event_id", envelope.EventID, "type", envelope.Type)
			}
			proc.discardEventResult(envelope.EventID)
			continue
		}
		if res != nil {
			results = append(results, res)
			if envelope.Type == "CHAT" && envelope.Chat != nil && res.Chat != nil {
				envelope.Chat.Message = res.Chat.Message
			}
		}
	}
	return results
}

func convertProtoDrops(drops []*proto.ItemStack) []item.Stack {
	if len(drops) == 0 {
		return nil
	}
	converted := make([]item.Stack, 0, len(drops))
	for _, drop := range drops {
		if drop == nil || drop.Name == "" {
			continue
		}
		material, ok := world.ItemByName(drop.Name, int16(drop.Meta))
		if !ok {
			continue
		}
		count := int(drop.Count)
		if count <= 0 {
			continue
		}
		converted = append(converted, item.NewStack(material, count))
	}
	return converted
}

func (m *Manager) handlePluginMessage(p *pluginProcess, msg *proto.PluginToHost) {
	if msg.Result != nil {
		p.deliverEventResult(msg.Result)
	}
	if msg.Hello != nil {
		p.setHello(msg.Hello)
		m.registerCommands(p, msg.Hello.Commands)
	}
	if msg.Subscribe != nil {
		p.updateSubscriptions(msg.Subscribe.Events)
	}
	if msg.Actions != nil {
		m.applyActions(p, msg.Actions)
	}
	if msg.Log != nil {
		level := strings.ToLower(msg.Log.Level)
		switch level {
		case "warn", "warning":
			p.log.Warn(msg.Log.Message)
		case "error":
			p.log.Error(msg.Log.Message)
		default:
			p.log.Info(msg.Log.Message)
		}
	}
}

func (m *Manager) registerCommands(p *pluginProcess, specs []*proto.CommandSpec) {
	for _, spec := range specs {
		if spec == nil || spec.Name == "" {
			continue
		}
		name := strings.TrimPrefix(spec.Name, "/")
		binding := commandBinding{pluginID: p.id, command: name, descriptor: spec}
		m.mu.Lock()
		m.commands[name] = binding
		m.mu.Unlock()
		cmd.Register(cmd.New(name, spec.Description, nil, pluginCommand{mgr: m, pluginID: p.id, name: name}))
	}
}

type pluginCommand struct {
	mgr      *Manager
	pluginID string
	name     string
}

func (c pluginCommand) Run(src cmd.Source, output *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok {
		output.Errorf("command only available to players")
		return
	}
	raw := "/" + c.name
	c.mgr.emitCommand(nil, p, raw)
	output.Printf("command forwarded to plugin")
}

func (m *Manager) applyActions(p *pluginProcess, batch *proto.ActionBatch) {
	if batch == nil {
		return
	}
	for _, action := range batch.Actions {
		if action == nil {
			continue
		}
		switch {
		case action.SendChat != nil:
			m.handleSendChat(action.SendChat)
		case action.Teleport != nil:
			m.handleTeleport(action.Teleport)
		case action.Kick != nil:
			m.handleKick(action.Kick)
		}
	}
}

func (m *Manager) handleSendChat(act *proto.SendChatAction) {
	if act.TargetUUID == "" {
		for p := range m.srv.Players(nil) {
			p.Message(act.Message)
		}
		chat.Global.WriteString(act.Message)
		return
	}
	id, err := uuid.Parse(act.TargetUUID)
	if err != nil {
		return
	}
	if handle, ok := m.srv.Player(id); ok {
		handle.ExecWorld(func(tx *world.Tx, e world.Entity) {
			if pl, ok := e.(*player.Player); ok {
				pl.Message(act.Message)
			}
		})
	}
}

func (m *Manager) handleTeleport(act *proto.TeleportAction) {
	id, err := uuid.Parse(act.PlayerUUID)
	if err != nil {
		return
	}
	if handle, ok := m.srv.Player(id); ok {
		handle.ExecWorld(func(tx *world.Tx, e world.Entity) {
			pl, ok := e.(*player.Player)
			if !ok {
				return
			}
			pl.Teleport(mgl64.Vec3{act.X, act.Y, act.Z})
			rot := pl.Rotation()
			deltaYaw := float64(act.Yaw) - rot.Yaw()
			deltaPitch := float64(act.Pitch) - rot.Pitch()
			pl.Move(mgl64.Vec3{}, deltaYaw, deltaPitch)
		})
	}
}

func (m *Manager) handleKick(act *proto.KickAction) {
	id, err := uuid.Parse(act.PlayerUUID)
	if err != nil {
		return
	}
	if handle, ok := m.srv.Player(id); ok {
		handle.ExecWorld(func(tx *world.Tx, e world.Entity) {
			if pl, ok := e.(*player.Player); ok {
				pl.Disconnect(act.Reason)
			}
		})
	}
}
