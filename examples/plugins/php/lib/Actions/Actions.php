<?php

namespace Dragonfly\PluginLib\Actions;

use Df\Plugin\Action;
use Df\Plugin\ActionBatch;
use Df\Plugin\PluginToHost;
use Df\Plugin\SendChatAction;
use Dragonfly\PluginLib\StreamSender;

final class Actions {
    public function __construct(
        private StreamSender $sender,
        private string $pluginId,
    ) {}

    public function sendAction(Action $action): void {
        $batch = new ActionBatch();
        $batch->setActions([$action]);

        $resp = new PluginToHost();
        $resp->setPluginId($this->pluginId);
        $resp->setActions($batch);
        $this->sender->enqueue($resp);
    }

    public function chatToUuid(string $targetUuid, string $message): void {
        $action = new Action();
        $send = new SendChatAction();
        $send->setTargetUuid($targetUuid);
        $send->setMessage($message);
        $action->setSendChat($send);
        $this->sendAction($action);
    }
}
