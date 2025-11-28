import { PluginBase } from '../plugin/PluginBase.js';
import { GameMode, Sound } from '../generated/common.js';

export class Player {
    constructor(private plugin: PluginBase, public readonly uuid: string) {}

    public sendMessage(message: string) {
        this.plugin.send({
            pluginId: this.plugin.pluginId,
            actions: {
                actions: [{
                    correlationId: `msg-${Date.now()}`,
                    sendChat: {
                        targetUuid: this.uuid,
                        message
                    }
                }]
            }
        });
    }

    public sendTitle(title: string, subtitle: string = '') {
         this.plugin.send({
            pluginId: this.plugin.pluginId,
            actions: {
                actions: [{
                    correlationId: `title-${Date.now()}`,
                    sendTitle: {
                        playerUuid: this.uuid,
                        title,
                        subtitle
                    }
                }]
            }
        });
    }

    public sendPopup(message: string) {
         this.plugin.send({
            pluginId: this.plugin.pluginId,
            actions: {
                actions: [{
                    correlationId: `popup-${Date.now()}`,
                    sendPopup: {
                        playerUuid: this.uuid,
                        message
                    }
                }]
            }
        });
    }

    public playSound(sound: Sound, volume: number = 1.0, pitch: number = 1.0) {
         this.plugin.send({
            pluginId: this.plugin.pluginId,
            actions: {
                actions: [{
                    correlationId: `sound-${Date.now()}`,
                    playSound: {
                        playerUuid: this.uuid,
                        sound,
                        volume,
                        pitch
                    }
                }]
            }
        });
    }

    public teleport(x: number, y: number, z: number, yaw: number = 0, pitch: number = 0) {
         this.plugin.send({
            pluginId: this.plugin.pluginId,
            actions: {
                actions: [{
                    correlationId: `tp-${Date.now()}`,
                    teleport: {
                        playerUuid: this.uuid,
                        position: { x, y, z },
                        rotation: { x: yaw, y: pitch, z: 0 }
                    }
                }]
            }
        });
    }

    public setGameMode(mode: GameMode) {
         this.plugin.send({
            pluginId: this.plugin.pluginId,
            actions: {
                actions: [{
                    correlationId: `gm-${Date.now()}`,
                    setGameMode: {
                        playerUuid: this.uuid,
                        gameMode: mode
                    }
                }]
            }
        });
    }

    public giveItem(name: string, count: number = 1, meta: number = 0) {
         this.plugin.send({
            pluginId: this.plugin.pluginId,
            actions: {
                actions: [{
                    correlationId: `give-${Date.now()}`,
                    giveItem: {
                        playerUuid: this.uuid,
                        item: {
                            name,
                            count,
                            meta
                        }
                    }
                }]
            }
        });
    }

    public clearInventory() {
        this.plugin.send({
            pluginId: this.plugin.pluginId,
            actions: {
                actions: [{
                    correlationId: `clear-${Date.now()}`,
                    clearInventory: {
                        playerUuid: this.uuid
                    }
                }]
            }
        });
    }
}
