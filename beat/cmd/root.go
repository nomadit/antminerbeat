package cmd

// reference source
// https://github.com/Pravoru/consulbeat
// https://github.com/christiangalsterer/execbeat
// https://github.com/christiangalsterer/httpbeat
// heartbeat

// reference library
// https://github.com/spf13/cobra

import (
	"github.com/elastic/beats/libbeat/cmd"
	"github.com/nomadit/antminerbeat/beat/beater"
)

// Name of this beat
var Name = "antminerbeat"

// RootCmd to handle beats cli
var RootCmd = cmd.GenRootCmd(Name, "", beater.New)
