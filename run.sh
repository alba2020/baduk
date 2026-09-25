#!/bin/bash
set -e

# Собираем сервер из его файлов в той же директории БЕЗ дублирования
go build -o play_atari play.go agent_model.go agent_logic.go
./play_atari
