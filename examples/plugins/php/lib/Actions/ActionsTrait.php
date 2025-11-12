<?php

namespace Dragonfly\PluginLib\Actions;

use Df\Plugin\Action;

trait ActionsTrait {
    abstract protected function getActions(): Actions;

    public function sendAction(Action $action): void {
        $this->getActions()->sendAction($action);
    }

    public function chatToUuid(string $targetUuid, string $message): void {
        $this->getActions()->chatToUuid($targetUuid, $message);
    }
}
