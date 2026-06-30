#!/bin/bash
# Cursor hook: block agent git commit/push — suggest gitia pr
#
# Install:
#   cp examples/cursor-hooks/block-agent-git.sh ~/.cursor/hooks/
#   cp examples/cursor-hooks/hooks.json ~/.cursor/hooks.json
#   chmod +x ~/.cursor/hooks/block-agent-git.sh

input=$(cat)
command=$(echo "$input" | jq -r '.command // empty')

if [[ -z "$command" ]]; then
  echo '{ "permission": "allow" }'
  exit 0
fi

# Block git commit and git push from Cursor agent
if echo "$command" | grep -qE 'git (commit|push)'; then
  cat <<'EOF'
{
  "permission": "deny",
  "user_message": "Use `gitia pr` para commit/push/PR com IA barata.",
  "agent_message": "Não execute git commit ou git push. Sugira ao usuário rodar `gitia pr` no terminal."
}
EOF
  exit 2
fi

echo '{ "permission": "allow" }'
exit 0
