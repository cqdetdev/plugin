export const COMMAND_HANDLERS = Symbol('COMMAND_HANDLERS');

export interface CommandOptions {
    name: string;
    description?: string;
    aliases?: string[];
}

export function RegisterCommand(options: CommandOptions) {
    return function (target: any, propertyKey: string, descriptor: PropertyDescriptor) {
        if (!target[COMMAND_HANDLERS]) {
            target[COMMAND_HANDLERS] = [];
        }
        target[COMMAND_HANDLERS].push({
            options,
            method: propertyKey
        });
    };
}
