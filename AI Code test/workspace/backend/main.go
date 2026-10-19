// Command claimconfig is the backend of the multi-tenant claim configuration
// platform (AI Challenge 15).
//
//	@title				Multi-Tenant Claim Config API
//	@version			1.0
//	@description		Tenant configuration (versioned, with diff and rollback) and runtime claim processing.
//	@description		Every endpoint answers HTTP 200 with {code, message, error_text, data[]}: the real status is in `code`, a stable error code in `message`, and the payload is `data[0]`.
//	@description		Authentication uses an HttpOnly session cookie. In this UI, run POST /auth/login first (e.g. admin@papaya.com / 123456); the browser then sends the cookie with every request.
//	@BasePath			/api/v1
//
//	@tag.name			health
//	@tag.description	Liveness
//	@tag.name			auth
//	@tag.description	Password login and sessions
//	@tag.name			tenants
//	@tag.description	Tenant CRUD, config history, rollback and diff
//	@tag.name			configs
//	@tag.description	Config enumerations, defaults and validation
//	@tag.name			claims
//	@tag.description	processClaim, preview and processed claims
package main

import (
	"os"

	"claimconfig/server"

	_ "time/tzdata" // tenant time zones work on Windows and in minimal containers
)

func main() {
	os.Exit(server.Main(os.Args[1:]))
}
