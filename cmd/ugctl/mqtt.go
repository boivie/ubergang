package main

import (
	"boivie/ubergang/server/api"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

func newMqttCmd(a *app) *cobra.Command {
	return groupCmd(&cobra.Command{
		Use:   "mqtt",
		Short: "Manage MQTT profiles and clients",
	},
		newMqttProfilesCmd(a),
		newMqttClientsCmd(a),
	)
}

func newMqttProfilesCmd(a *app) *cobra.Command {
	return groupCmd(&cobra.Command{
		Use:     "profiles",
		Aliases: []string{"profile"},
		Short:   "Manage MQTT profiles, which set the topics that clients may use",
		Long: `Manage MQTT profiles, which set the topics that clients may publish and
subscribe to.

Topics can use $ID for the client's ID, and $<key> for any of the client's
values, such as "sensors/$ID/#" or "rooms/$room/#".`,
	},
		newMqttProfilesListCmd(a),
		newMqttProfilesGetCmd(a),
		newMqttProfilesCreateCmd(a),
		newMqttProfilesUpdateCmd(a),
		newMqttProfilesDeleteCmd(a),
	)
}

func newMqttProfilesListCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List MQTT profiles",
		Args:    exactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.newClient()
			if err != nil {
				return err
			}
			res, err := c.listMqttProfiles(cmd.Context())
			if err != nil {
				return err
			}
			if a.output == outputJSON {
				return a.printJSON(res)
			}
			if len(res.Profiles) == 0 {
				_, err := fmt.Fprintln(a.stderr, "No MQTT profiles.")
				return err
			}
			rows := make([][]string, 0, len(res.Profiles))
			for _, p := range res.Profiles {
				rows = append(rows, []string{p.Id, joinOrDash(p.AllowPublish), joinOrDash(p.AllowSubscribe)})
			}
			return a.printTable([]string{"ID", "PUBLISH", "SUBSCRIBE"}, rows)
		},
	}
}

func newMqttProfilesGetCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show an MQTT profile",
		Args:  idArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.newClient()
			if err != nil {
				return err
			}
			p, err := c.getMqttProfile(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return a.printMqttProfile(p)
		},
	}
}

func newMqttProfilesCreateCmd(a *app) *cobra.Command {
	var publish, subscribe []string
	cmd := &cobra.Command{
		Use:     "create <id>",
		Short:   "Create an MQTT profile",
		Example: `  ugctl mqtt profiles create sensors --publish 'sensors/$ID/#' --subscribe 'commands/$ID'`,
		Args:    idArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.newClient()
			if err != nil {
				return err
			}
			p, err := c.createMqttProfile(cmd.Context(), &api.PublicCreateMqttProfileRequest{
				Id:             args[0],
				AllowPublish:   orEmpty(publish),
				AllowSubscribe: orEmpty(subscribe),
			})
			if err != nil {
				return err
			}
			return a.printMqttProfile(p)
		},
	}
	cmd.Flags().StringArrayVar(&publish, "publish", nil, "a `topic` that clients may publish to (repeatable)")
	cmd.Flags().StringArrayVar(&subscribe, "subscribe", nil, "a `topic` that clients may subscribe to (repeatable)")
	return cmd
}

func newMqttProfilesUpdateCmd(a *app) *cobra.Command {
	var publish, subscribe []string
	var clearPublish, clearSubscribe bool
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Change an MQTT profile",
		Long: `Changes an MQTT profile. The given topics replace the existing ones, and
topics that aren't given are left unchanged.

Clients that use the profile are disconnected, so that they reconnect with the
new permissions.`,
		Example: `  ugctl mqtt profiles update sensors --publish 'sensors/$ID/#' --publish 'rooms/$room/#'
  ugctl mqtt profiles update sensors --clear-subscribe`,
		Args: idArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			req := api.PublicUpdateMqttProfileRequest{}
			var err error
			if req.AllowPublish, err = topicsChange(cmd, "publish", publish, clearPublish); err != nil {
				return err
			}
			if req.AllowSubscribe, err = topicsChange(cmd, "subscribe", subscribe, clearSubscribe); err != nil {
				return err
			}
			if req.AllowPublish == nil && req.AllowSubscribe == nil {
				return usageErrorf("nothing to change: use --publish, --subscribe, --clear-publish or --clear-subscribe")
			}
			c, err := a.newClient()
			if err != nil {
				return err
			}
			p, err := c.updateMqttProfile(cmd.Context(), args[0], &req)
			if err != nil {
				return err
			}
			return a.printMqttProfile(p)
		},
	}
	cmd.Flags().StringArrayVar(&publish, "publish", nil, "a `topic` that clients may publish to (repeatable)")
	cmd.Flags().StringArrayVar(&subscribe, "subscribe", nil, "a `topic` that clients may subscribe to (repeatable)")
	cmd.Flags().BoolVar(&clearPublish, "clear-publish", false, "don't allow publishing to any topic")
	cmd.Flags().BoolVar(&clearSubscribe, "clear-subscribe", false, "don't allow subscribing to any topic")
	return cmd
}

// topicsChange returns the new topics for "--<name>" and "--clear-<name>", or
// nil if they should be left unchanged.
func topicsChange(cmd *cobra.Command, name string, topics []string, clear bool) (*[]string, error) {
	set := cmd.Flags().Changed(name)
	switch {
	case set && clear:
		return nil, usageErrorf("--%s and --clear-%s can't be used together", name, name)
	case set:
		return &topics, nil
	case clear:
		return &[]string{}, nil
	}
	return nil, nil
}

func newMqttProfilesDeleteCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm"},
		Short:   "Delete an MQTT profile",
		Long:    "Deletes an MQTT profile. This fails if any client still uses it.",
		Args:    idArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.newClient()
			if err != nil {
				return err
			}
			if err := c.deleteMqttProfile(cmd.Context(), args[0]); err != nil {
				return err
			}
			if a.output == outputJSON {
				return nil
			}
			return a.printMessage("Deleted MQTT profile %s.", args[0])
		},
	}
}

// idArg checks that a command is given a single resource ID. IDs are used in
// URLs, so they can't be empty or contain '/'.
func idArg(cmd *cobra.Command, args []string) error {
	if err := exactArgs(1)(cmd, args); err != nil {
		return err
	}
	if args[0] == "" || strings.Contains(args[0], "/") {
		return usageErrorf("invalid ID %q: it can't be empty or contain '/'", args[0])
	}
	return nil
}

func (a *app) printMqttProfile(p *api.PublicMqttProfile) error {
	if a.output == outputJSON {
		return a.printJSON(p)
	}
	return a.printFields([]field{
		{"ID", []string{p.Id}},
		{"Publish", p.AllowPublish},
		{"Subscribe", p.AllowSubscribe},
	})
}

func newMqttClientsCmd(a *app) *cobra.Command {
	return groupCmd(&cobra.Command{
		Use:     "clients",
		Aliases: []string{"client"},
		Short:   "Manage MQTT clients, such as devices, and their passwords",
	},
		newMqttClientsListCmd(a),
		newMqttClientsGetCmd(a),
		newMqttClientsCreateCmd(a),
		newMqttClientsUpdateCmd(a),
		newMqttClientsDeleteCmd(a),
	)
}

func newMqttClientsListCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List MQTT clients",
		Long: `Lists MQTT clients. Passwords are left out of the table, but are included
in JSON output for API keys with the write scope.`,
		Args: exactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.newClient()
			if err != nil {
				return err
			}
			res, err := c.listMqttClients(cmd.Context())
			if err != nil {
				return err
			}
			if a.output == outputJSON {
				return a.printJSON(res)
			}
			if len(res.Clients) == 0 {
				_, err := fmt.Fprintln(a.stderr, "No MQTT clients.")
				return err
			}
			rows := make([][]string, 0, len(res.Clients))
			for _, cl := range res.Clients {
				rows = append(rows, []string{cl.Id, cl.ProfileId, joinOrDash(formatValues(cl.Values))})
			}
			return a.printTable([]string{"ID", "PROFILE", "VALUES"}, rows)
		},
	}
}

func newMqttClientsGetCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show an MQTT client",
		Long:  "Shows an MQTT client. The password is only shown for API keys with the write scope.",
		Args:  idArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.newClient()
			if err != nil {
				return err
			}
			cl, err := c.getMqttClient(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return a.printMqttClient(cl)
		},
	}
}

// passwordFlags are the ways to give an MQTT client's password.
type passwordFlags struct {
	password string
	stdin    bool
	generate bool
}

func (f *passwordFlags) register(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.password, "password", "",
		"the password; prefer --password-stdin, as this is visible to other local users")
	cmd.Flags().BoolVar(&f.stdin, "password-stdin", false, "read the password from stdin")
	cmd.Flags().BoolVar(&f.generate, "generate-password", false, "generate a random password, which is printed")
}

// value returns the password, or nil if none of the flags were given.
func (f *passwordFlags) value(cmd *cobra.Command, stdin io.Reader) (*string, error) {
	given := 0
	for _, name := range []string{"password", "password-stdin", "generate-password"} {
		if cmd.Flags().Changed(name) {
			given++
		}
	}
	if given > 1 {
		return nil, usageErrorf("only one of --password, --password-stdin and --generate-password can be used")
	}
	var password string
	switch {
	case cmd.Flags().Changed("password"):
		password = f.password
	case f.stdin:
		var err error
		if password, err = readLine(stdin); err != nil {
			return nil, err
		}
	case f.generate:
		buf := make([]byte, 16)
		if _, err := rand.Read(buf); err != nil {
			return nil, err
		}
		password = hex.EncodeToString(buf)
	default:
		return nil, nil
	}
	if password == "" {
		return nil, usageErrorf("the password can't be empty")
	}
	return &password, nil
}

// parseValues parses "key=value" flags.
func parseValues(flags []string) (map[string]string, error) {
	values := make(map[string]string, len(flags))
	for _, f := range flags {
		k, v, ok := strings.Cut(f, "=")
		if !ok || k == "" {
			return nil, usageErrorf("invalid --value %q: must be key=value", f)
		}
		if _, dup := values[k]; dup {
			return nil, usageErrorf("--value %q is given more than once", k)
		}
		values[k] = v
	}
	return values, nil
}

func newMqttClientsCreateCmd(a *app) *cobra.Command {
	var profile string
	var values []string
	var pw passwordFlags
	cmd := &cobra.Command{
		Use:   "create <id>",
		Short: "Create an MQTT client",
		Long: `Creates an MQTT client. It needs a profile and a password, given with
--password-stdin, --generate-password or --password.`,
		Example: `  ugctl mqtt clients create dev-1 --profile sensors --generate-password --value room=kitchen
  openssl rand -hex 16 | ugctl mqtt clients create dev-1 --profile sensors --password-stdin`,
		Args: idArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			if profile == "" {
				return usageErrorf("--profile is required")
			}
			password, err := pw.value(cmd, a.stdin)
			if err != nil {
				return err
			}
			if password == nil {
				return usageErrorf("a password is required: use --password-stdin, --generate-password or --password")
			}
			vals, err := parseValues(values)
			if err != nil {
				return err
			}
			c, err := a.newClient()
			if err != nil {
				return err
			}
			cl, err := c.createMqttClient(cmd.Context(), &api.PublicCreateMqttClientRequest{
				Id:        args[0],
				ProfileId: profile,
				Password:  *password,
				Values:    vals,
			})
			if err != nil {
				return err
			}
			return a.printMqttClient(cl)
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "", "the client's MQTT `profile` (required)")
	cmd.Flags().StringArrayVar(&values, "value", nil, "a `key=value` pair, usable in the profile's topics as $key (repeatable)")
	pw.register(cmd)
	return cmd
}

func newMqttClientsUpdateCmd(a *app) *cobra.Command {
	var profile string
	var values []string
	var clearValues bool
	var pw passwordFlags
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Change an MQTT client",
		Long: `Changes an MQTT client. Anything that isn't given is left unchanged, but
--value replaces all of the client's values.

Changing the profile or password disconnects the client.`,
		Example: `  ugctl mqtt clients update dev-1 --generate-password
  ugctl mqtt clients update dev-1 --profile actuators --value room=hall`,
		Args: idArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			req := api.PublicUpdateMqttClientRequest{}
			if cmd.Flags().Changed("profile") {
				req.ProfileId = &profile
			}
			var err error
			if req.Password, err = pw.value(cmd, a.stdin); err != nil {
				return err
			}
			switch {
			case cmd.Flags().Changed("value") && clearValues:
				return usageErrorf("--value and --clear-values can't be used together")
			case cmd.Flags().Changed("value"):
				vals, err := parseValues(values)
				if err != nil {
					return err
				}
				req.Values = &vals
			case clearValues:
				req.Values = &map[string]string{}
			}
			if req.ProfileId == nil && req.Password == nil && req.Values == nil {
				return usageErrorf("nothing to change: use --profile, a password flag, --value or --clear-values")
			}
			c, err := a.newClient()
			if err != nil {
				return err
			}
			cl, err := c.updateMqttClient(cmd.Context(), args[0], &req)
			if err != nil {
				return err
			}
			return a.printMqttClient(cl)
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "", "the client's new MQTT `profile`")
	cmd.Flags().StringArrayVar(&values, "value", nil, "a `key=value` pair, replacing all existing values (repeatable)")
	cmd.Flags().BoolVar(&clearValues, "clear-values", false, "remove all values")
	pw.register(cmd)
	return cmd
}

func newMqttClientsDeleteCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm"},
		Short:   "Delete an MQTT client",
		Long:    "Deletes an MQTT client and disconnects it.",
		Args:    idArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := a.newClient()
			if err != nil {
				return err
			}
			if err := c.deleteMqttClient(cmd.Context(), args[0]); err != nil {
				return err
			}
			if a.output == outputJSON {
				return nil
			}
			return a.printMessage("Deleted MQTT client %s.", args[0])
		},
	}
}

func (a *app) printMqttClient(cl *api.PublicMqttClient) error {
	if a.output == outputJSON {
		return a.printJSON(cl)
	}
	fields := []field{
		{"ID", []string{cl.Id}},
		{"Profile", []string{cl.ProfileId}},
	}
	if cl.Password != "" {
		fields = append(fields, field{"Password", []string{cl.Password}})
	}
	fields = append(fields, field{"Values", formatValues(cl.Values)})
	return a.printFields(fields)
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
