#!/bin/sh
set -e
chown -R tsunagu:tsunagu /app/data /app/jar-cache /app/sandbox/extensions
exec gosu tsunagu /app/tsunagu-server
