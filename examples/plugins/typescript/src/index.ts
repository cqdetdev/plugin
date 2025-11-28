import {
    PluginBase,
    On,
    EventType,
    PlayerMoveEvent,
    PlayerQuitEvent,
    PlayerJoinEvent,
    EventContext,
    Player,
    Sound,
    RegisterCommand
} from '@dragonfly/proto';

class AreaPlugin extends PluginBase {
    // Define a restricted area: x > 5
    private readonly RESTRICTED_X = 5;
    private readonly VIEW_DISTANCE = 25;
    private readonly WALL_WIDTH = 20;  // Radius along Z
    private readonly WALL_HEIGHT = 10; // Radius along Y

    // Track active wall blocks per player to minimize updates
    // UUID -> Set<"x,y,z">
    private activeWalls: Map<string, Set<string>> = new Map();

    onLoad(): void {
        console.log('[AreaPlugin] Plugin loaded.');
    }

    onEnable(): void {
        console.log('[AreaPlugin] Enabled. Dynamic wall active at X = 5.');
    }

    onDisable(): void {
        console.log('[AreaPlugin] Disabled.');
    }

    @On(EventType.PLAYER_MOVE)
    onMove(event: PlayerMoveEvent, context: EventContext<PlayerMoveEvent>) {
        const pos = event.position;
        if (!pos) {
            context.ack();
            return;
        }

        const player = new Player(this, event.playerUuid);

        // 1. Enforcement: Prevent crossing X > 5
        if (pos.x > this.RESTRICTED_X) {
            // Cancel movement
            //context.cancel();
            player.sendPopup('§cRestricted Area! Turn back.');
            //player.playSound(Sound.ITEM_BREAK);
        } else {
        }

        // 2. Visuals: Dynamic Wall
        this.updateWall(event.playerUuid, pos);
                    context.ack();

    }
    @On(EventType.PLAYER_QUIT)
    onQuit(event: PlayerQuitEvent, context: EventContext<PlayerQuitEvent>) {
        // Clean up state
        const active = this.activeWalls.get(event.playerUuid);
        if (active) {
            this.activeWalls.delete(event.playerUuid);
        }
        context.ack();
    }

    @On(EventType.PLAYER_JOIN)
    onJoin(event: PlayerJoinEvent, context: EventContext<PlayerJoinEvent>) {
        const player = new Player(this, event.playerUuid);
        // Teleport player near the wall area (0, -60, 0)
        player.teleport(10, -60, 10); 
        player.sendMessage('§aWelcome! Wall is at X = ' + this.RESTRICTED_X + '.');
        context.ack();
    }

    @RegisterCommand({ name: 'wall', description: 'Build a glass wall at the border' })
    onWallCommand(uuid: string, args: string[], context: EventContext<any>) {
        const player = new Player(this, uuid);
        player.sendMessage('§eBuilding wall...');

        const actions: any[] = []; 
        // Build a wall at x=5, from z=-5 to 5, y=-60 to -55
        for (let z = -5; z <= 5; z++) {
            for (let y = -60; y <= -55; y++) {
                actions.push({
                    worldSetBlock: {
                        world: { name: '', dimension: 'overworld', id: '' }, 
                        position: { x: this.RESTRICTED_X, y: y, z: z },
                        block: { name: 'minecraft:glass' }
                    }
                });
            }
        }

        this.send({
            pluginId: this.pluginId,
            actions: {
                actions: actions
            }
        });
        
        player.sendMessage('§aWall built!');
    }

    private updateWall(uuid: string, playerPos: { x: number, y: number, z: number }) {
        const dist = Math.abs(playerPos.x - this.RESTRICTED_X);
        const desiredBlocks = new Set<string>();

        // Only render if close to the wall
        if (dist <= this.VIEW_DISTANCE) {
            const centerY = Math.round(playerPos.y);
            const centerZ = Math.round(playerPos.z);

            for (let y = centerY - this.WALL_HEIGHT; y <= centerY + this.WALL_HEIGHT; y++) {
                for (let z = centerZ - this.WALL_WIDTH; z <= centerZ + this.WALL_WIDTH; z++) {
                    // Wall is a plane at RESTRICTED_X
                    desiredBlocks.add(`${this.RESTRICTED_X},${y},${z}`);
                }
            }
        }

        // Get current state
        let currentBlocks = this.activeWalls.get(uuid);
        if (!currentBlocks) {
            currentBlocks = new Set();
            this.activeWalls.set(uuid, currentBlocks);
        }

        // Calculate diff
        const toAdd: string[] = [];
        const toRemove: string[] = [];

        for (const block of desiredBlocks) {
            if (!currentBlocks.has(block)) toAdd.push(block);
        }
        for (const block of currentBlocks) {
            if (!desiredBlocks.has(block)) toRemove.push(block);
        }

        // If no changes, active state is stable
        if (toAdd.length === 0 && toRemove.length === 0) {
            return;
        }

        const actions: any[] = [];

        // Add new glass blocks
        for (const key of toAdd) {
            const [x, y, z] = key.split(',').map(Number);
            actions.push({
                worldSetBlock: {
                    world: { name: '', dimension: 'overworld', id: '' },
                    position: { x, y, z },
                    block: { name: 'minecraft:glass'}
                }
            });
            currentBlocks.add(key);
        }

        // Remove old blocks (restore to air - simplified)
        for (const key of toRemove) {
            const [x, y, z] = key.split(',').map(Number);
            actions.push({
                worldSetBlock: {
                    world: { name: '', dimension: 'overworld', id: '' },
                    position: { x, y, z },
                    block: { name: 'minecraft:air', properties: {} }
                }
            });
            currentBlocks.delete(key);
        }

        // Batch send updates
        this.send({
            pluginId: this.pluginId,
            actions: {
                actions: actions
            }
        });
    }
}

new AreaPlugin().run();
