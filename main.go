package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/justinas/alice"
	"github.com/mileusna/useragent"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"

	"github.com/alecthomas/kong"
)

type Globals struct {
	LogLevel  string      `short:"l" help:"Set the logging level (debug|info|warn|error|fatal)" default:"info" env:"LOG_LEVEL"`
	LogFormat string      `short:"f" help:"Set the logging format (text|json)" default:"text" env:"LOG_FORMAT"`
	Version   VersionFlag `name:"version" help:"Print version information and quit"`
	Color     string      `short:"c" help:"Color to be displayed" env:"COLOR" required:""`
}

type VersionFlag string

func (v VersionFlag) Decode(ctx *kong.DecodeContext) error { return nil }
func (v VersionFlag) IsBool() bool                         { return true }
func (v VersionFlag) BeforeApply(app *kong.Kong, vars kong.Vars) error {
	fmt.Println(vars["version"])
	app.Exit(0)
	return nil
}

type CLI struct {
	Globals
}

func serve(color string) {

	// Add access logs in zerologs
	chain := alice.New()
	// Install the logger handler with default output on the console
	chain = chain.Append(hlog.NewHandler(log.Logger))
	// Install some provided extra handler to set some request's context fields.
	// Thanks to that handler, all our logs will come with some prepopulated fields.
	chain = chain.Append(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).Info().
			Str("method", r.Method).
			Stringer("url", r.URL).
			Int("status", status).
			Int("size", size).
			Dur("duration", duration).
			Msg("")
	}))
	chain = chain.Append(hlog.RemoteAddrHandler("ip"))
	chain = chain.Append(hlog.UserAgentHandler("user_agent"))
	chain = chain.Append(hlog.RefererHandler("referer"))
	chain = chain.Append(hlog.RequestIDHandler("req_id", "Request-Id"))

	rootHandler := chain.Then(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				answer := strings.ToUpper(color)
				ua := useragent.Parse(r.UserAgent())
				if ua.IsChrome() || ua.IsFirefox() || ua.IsOpera() || ua.IsEdge() || ua.IsSafari() || ua.IsInternetExplorer() {
					answer = fmt.Sprintf(
						`<!DOCTYPE html>
						<html>
						<head>
							<title>%s</title>
							<style>
							body {
								background-color: %s;
							}
							.middle {
								left: 0;
								line-height: 200px;
								margin-top: -100px;
								position: absolute;
								text-align: center;
								top: 50%%;
								width: 100%%;
							}
							</style>
						</head>
						<body><div class="middle">%s</div></body>
						</html>`,
						color,
						color,
						answer)
				}
				fmt.Fprint(w, answer)
			}))

	http.Handle("/", rootHandler)

	log.Info().Msg("Listening on port 3000...")
	if err := http.ListenAndServe(":3000", nil); err != nil {
		log.Fatal().Msg(err.Error())
	}

}

func main() {
	cli := CLI{
		Globals: Globals{
			Version: VersionFlag("1.0.0"),
		},
	}

	ctx := kong.Parse(&cli,
		kong.Name("rainbow"),
		kong.Description("Rainbow answer a simple HTML page with background of the color specified in the env var."),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Vars{
			"version": string(cli.Globals.Version),
		})

	if strings.ToLower(cli.Globals.LogFormat) == "text" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
			With().
			Timestamp().
			Caller().
			Logger()
	}

	switch strings.ToLower(cli.Globals.LogLevel) {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "panic":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	}

	switch ctx.Command() {
	default:
		serve(cli.Globals.Color)
	}
}
