#!/bin/env zsh

TOP_LEFT=$(qdbus6 org.kde.yakuake /yakuake/sessions org.kde.yakuake.activeTerminalId)
BOTTOM_LEFT=$(qdbus6 org.kde.yakuake /yakuake/sessions org.kde.yakuake.splitTerminalTopBottom $TOP_LEFT)
TOP_RIGHT=$(qdbus6 org.kde.yakuake /yakuake/sessions org.kde.yakuake.splitTerminalLeftRight $TOP_LEFT)
BOTTOM_RIGHT=$(qdbus6 org.kde.yakuake /yakuake/sessions org.kde.yakuake.splitTerminalLeftRight $BOTTOM_LEFT)

qdbus6 org.kde.yakuake /yakuake/sessions org.kde.yakuake.runCommandInTerminal $TOP_LEFT "./dev/watchexec.sh worker"
qdbus6 org.kde.yakuake /yakuake/sessions org.kde.yakuake.runCommandInTerminal $TOP_RIGHT "temporal server start-dev --ui-port 8080"
qdbus6 org.kde.yakuake /yakuake/sessions org.kde.yakuake.runCommandInTerminal $BOTTOM_LEFT "./dev/watchexec.sh server"
qdbus6 org.kde.yakuake /yakuake/sessions org.kde.yakuake.runCommandInTerminal $BOTTOM_RIGHT "echo 'Commands here!'"