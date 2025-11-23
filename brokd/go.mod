module github.com/crolbar/brok/brokd

go 1.24.4

require (
	github.com/crolbar/brok/share v0.0.0-00010101000000-000000000000
	github.com/godbus/dbus v4.1.0+incompatible
)

replace github.com/crolbar/brok/share => ../share
