#!/bin/bash
set -e

# Собираем тренера только из его файлов
go build -o train_atari train.go match.go agent_model.go agent_logic.go
./train_atari
