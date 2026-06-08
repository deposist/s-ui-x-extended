#!/bin/sh

if [ "${SUI_MIGRATE_ONLY:-0}" = "1" ]; then
	exec ./sui migrate
fi

exec ./sui
