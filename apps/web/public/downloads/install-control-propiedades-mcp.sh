#!/bin/sh
set -eu

install_root="${XDG_DATA_HOME:-$HOME/.local/share}/control-propiedades"
config_root="${XDG_CONFIG_HOME:-$HOME/.config}/control-propiedades"
bin_root="$HOME/.local/bin"

mkdir -p "$install_root" "$config_root" "$bin_root"
curl -fsSL "https://control-propiedades-web.vercel.app/downloads/control-propiedades-mcp.mjs" -o "$install_root/server.mjs"
chmod 700 "$install_root/server.mjs"

launcher="$bin_root/control-propiedades-mcp"
printf '%s\n' '#!/bin/sh' 'set -eu' 'config_file="${XDG_CONFIG_HOME:-$HOME/.config}/control-propiedades/agent.env"' 'if [ ! -f "$config_file" ]; then echo "Falta $config_file" >&2; exit 1; fi' 'set -a' '. "$config_file"' 'set +a' 'exec node "${XDG_DATA_HOME:-$HOME/.local/share}/control-propiedades/server.mjs"' > "$launcher"
chmod 700 "$launcher"

echo "MCP instalado en $launcher"
echo "Guarda la credencial en $config_root/agent.env y luego regístralo en Hermes."
