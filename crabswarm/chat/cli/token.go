package cli

import (
	"fmt"
	"os"
)

// TokenEnvVar carries the identity token a human presents to the chat broker —
// the one `crabswarm chat admin register` printed. A human has no cmdman command to
// be recognized by, so this is how the token reaches the CLI when it is not
// passed as a flag.
const TokenEnvVar = "CRABSWARM_CHAT_TOKEN"

// CmdIDEnvVar carries the cmdman command id of the process the CLI runs in.
// cmdman sets it for every command it manages, so an agent inherits its own
// identity token without anyone configuring it; the daemon resolves it through
// the same cmdman that issued it.
const CmdIDEnvVar = "CMDMAN_CMD_ID"

// ResolveToken picks the caller's identity token: the --token flag first, then
// $CRABSWARM_CHAT_TOKEN, then $CMDMAN_CMD_ID. The flag wins so a human can act
// as a registered member from inside a cmdman-managed shell, which would
// otherwise hand them the shell's own identity.
func ResolveToken(flagToken string) (string, error) {
	return resolveToken(flagToken, os.LookupEnv)
}

// resolveToken is [ResolveToken] with the environment injected, so the
// precedence is testable without touching the process environment.
func resolveToken(
	flagToken string,
	lookupEnv func(string) (string, bool),
) (string, error) {
	if flagToken != "" {
		return flagToken, nil
	}
	for _, name := range []string{TokenEnvVar, CmdIDEnvVar} {
		if v, ok := lookupEnv(name); ok && v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf(
		"no chat identity token: pass --token, or set $%s to the token "+
			"`crabswarm chat admin register` printed; an agent under cmdman inherits $%s",
		TokenEnvVar, CmdIDEnvVar)
}
