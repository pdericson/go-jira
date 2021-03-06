package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"syscall"

	"github.com/coryb/figtree"
	"github.com/coryb/kingpeon"
	"github.com/coryb/oreo"

	jira "gopkg.in/Netflix-Skunkworks/go-jira.v1"
	"gopkg.in/Netflix-Skunkworks/go-jira.v1/jiracli"
	"gopkg.in/Netflix-Skunkworks/go-jira.v1/jiracmd"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/op/go-logging.v1"
)

var (
	log           = logging.MustGetLogger("jira")
	defaultFormat = func() string {
		format := os.Getenv("JIRA_LOG_FORMAT")
		if format != "" {
			return format
		}
		return "%{color}%{level:-5s}%{color:reset} %{message}"
	}()
)

func handleExit() {
	if e := recover(); e != nil {
		if exit, ok := e.(jiracli.Exit); ok {
			os.Exit(exit.Code)
		} else {
			fmt.Fprintf(os.Stderr, "%s\n%s", e, debug.Stack())
			os.Exit(1)
		}
	}
}

func increaseLogLevel(verbosity int) {
	logging.SetLevel(logging.GetLevel("")+logging.Level(verbosity), "")
	if logging.GetLevel("") > logging.DEBUG {
		oreo.TraceRequestBody = true
		oreo.TraceResponseBody = true
	}
}

func main() {
	defer handleExit()
	logBackend := logging.NewLogBackend(os.Stderr, "", 0)
	format := os.Getenv("JIRA_LOG_FORMAT")
	if format == "" {
		format = defaultFormat
	}
	logging.SetBackend(
		logging.NewBackendFormatter(
			logBackend,
			logging.MustStringFormatter(format),
		),
	)
	if os.Getenv("JIRA_DEBUG") == "" {
		logging.SetLevel(logging.NOTICE, "")
	} else {
		logging.SetLevel(logging.DEBUG, "")
	}

	app := kingpin.New("jira", "Jira Command Line Interface")
	app.Command("version", "Prints version").PreAction(func(*kingpin.ParseContext) error {
		fmt.Println(jira.VERSION)
		panic(jiracli.Exit{Code: 0})
	})

	var verbosity int
	app.Flag("verbose", "Increase verbosity for debugging").Short('v').PreAction(func(_ *kingpin.ParseContext) error {
		os.Setenv("JIRA_DEBUG", fmt.Sprintf("%d", verbosity))
		increaseLogLevel(1)
		return nil
	}).CounterVar(&verbosity)

	if os.Getenv("JIRA_DEBUG") != "" {
		if verbosity, err := strconv.Atoi(os.Getenv("JIRA_DEBUG")); err == nil {
			increaseLogLevel(verbosity)
		}
	}

	fig := figtree.NewFigTree()
	fig.EnvPrefix = "JIRA"
	fig.ConfigDir = ".jira.d"

	if err := os.MkdirAll(filepath.Join(jiracli.Homedir(), fig.ConfigDir), 0755); err != nil {
		log.Errorf("%s", err)
		panic(jiracli.Exit{Code: 1})
	}

	o := oreo.New().WithCookieFile(filepath.Join(jiracli.Homedir(), fig.ConfigDir, "cookies.js"))

	registry := []jiracli.CommandRegistry{
		jiracli.CommandRegistry{
			Command: "login",
			Entry:   jiracmd.CmdLoginRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "logout",
			Entry:   jiracmd.CmdLogoutRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "list",
			Aliases: []string{"ls"},
			Entry:   jiracmd.CmdListRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "view",
			Entry:   jiracmd.CmdViewRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "create",
			Entry:   jiracmd.CmdCreateRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "edit",
			Entry:   jiracmd.CmdEditRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "comment",
			Entry:   jiracmd.CmdCommentRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "worklog list",
			Entry:   jiracmd.CmdWorklogListRegistry(),
			Default: true,
		},
		jiracli.CommandRegistry{
			Command: "worklog add",
			Entry:   jiracmd.CmdWorklogAddRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "fields",
			Entry:   jiracmd.CmdFieldsRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "createmeta",
			Entry:   jiracmd.CmdCreateMetaRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "editmeta",
			Entry:   jiracmd.CmdEditMetaRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "subtask",
			Entry:   jiracmd.CmdSubtaskRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "dup",
			Entry:   jiracmd.CmdDupRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "block",
			Entry:   jiracmd.CmdBlockRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "issuelink",
			Entry:   jiracmd.CmdIssueLinkRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "issuelinktypes",
			Entry:   jiracmd.CmdIssueLinkTypesRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "transition",
			Aliases: []string{"trans"},
			Entry:   jiracmd.CmdTransitionRegistry(""),
		},
		jiracli.CommandRegistry{
			Command: "transitions",
			Entry:   jiracmd.CmdTransitionsRegistry("transitions"),
		},
		jiracli.CommandRegistry{
			Command: "transmeta",
			Entry:   jiracmd.CmdTransitionsRegistry("debug"),
		},
		jiracli.CommandRegistry{
			Command: "close",
			Entry:   jiracmd.CmdTransitionRegistry("close"),
		},
		jiracli.CommandRegistry{
			Command: "acknowledge",
			Aliases: []string{"ack"},
			Entry:   jiracmd.CmdTransitionRegistry("acknowledge"),
		},
		jiracli.CommandRegistry{
			Command: "reopen",
			Entry:   jiracmd.CmdTransitionRegistry("reopen"),
		},
		jiracli.CommandRegistry{
			Command: "resolve",
			Entry:   jiracmd.CmdTransitionRegistry("resolve"),
		},
		jiracli.CommandRegistry{
			Command: "start",
			Entry:   jiracmd.CmdTransitionRegistry("start"),
		},
		jiracli.CommandRegistry{
			Command: "stop",
			Entry:   jiracmd.CmdTransitionRegistry("stop"),
		},
		jiracli.CommandRegistry{
			Command: "todo",
			Entry:   jiracmd.CmdTransitionRegistry("To Do"),
		},
		jiracli.CommandRegistry{
			Command: "backlog",
			Entry:   jiracmd.CmdTransitionRegistry("Backlog"),
		},
		jiracli.CommandRegistry{
			Command: "done",
			Entry:   jiracmd.CmdTransitionRegistry("Done"),
		},
		jiracli.CommandRegistry{
			Command: "in-progress",
			Aliases: []string{"prog", "progress"},
			Entry:   jiracmd.CmdTransitionRegistry("Progress"),
		},
		jiracli.CommandRegistry{
			Command: "vote",
			Entry:   jiracmd.CmdVoteRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "rank",
			Entry:   jiracmd.CmdRankRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "watch",
			Entry:   jiracmd.CmdWatchRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "labels add",
			Entry:   jiracmd.CmdLabelsAddRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "labels set",
			Entry:   jiracmd.CmdLabelsSetRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "labels remove",
			Entry:   jiracmd.CmdLabelsRemoveRegistry(),
			Aliases: []string{"rm"},
		},
		jiracli.CommandRegistry{
			Command: "take",
			Entry:   jiracmd.CmdTakeRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "assign",
			Entry:   jiracmd.CmdAssignRegistry(),
			Aliases: []string{"give"},
		},
		jiracli.CommandRegistry{
			Command: "unassign",
			Entry:   jiracmd.CmdUnassignRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "component add",
			Entry:   jiracmd.CmdComponentAddRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "components",
			Entry:   jiracmd.CmdComponentsRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "issuetypes",
			Entry:   jiracmd.CmdIssueTypesRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "export-templates",
			Entry:   jiracmd.CmdExportTemplatesRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "unexport-templates",
			Entry:   jiracmd.CmdUnexportTemplatesRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "browse",
			Entry:   jiracmd.CmdBrowseRegistry(),
			Aliases: []string{"b"},
		},
		jiracli.CommandRegistry{
			Command: "request",
			Entry:   jiracmd.CmdRequestRegistry(),
			Aliases: []string{"req"},
		},
	}

	jiracli.Register(app, o, fig, registry)

	// register custom commands
	data := struct {
		CustomCommands kingpeon.DynamicCommands `yaml:"custom-commands" json:"custom-commands"`
	}{}

	if err := fig.LoadAllConfigs("config.yml", &data); err != nil {
		log.Errorf("%s", err)
		panic(jiracli.Exit{Code: 1})
	}

	if len(data.CustomCommands) > 0 {
		runner := syscall.Exec
		if runtime.GOOS == "windows" {
			runner = func(binary string, cmd []string, env []string) error {
				command := exec.Command(binary, cmd[1:]...)
				command.Stdin = os.Stdin
				command.Stdout = os.Stdout
				command.Stderr = os.Stderr
				command.Env = env
				return command.Run()
			}
		}

		tmp := map[string]interface{}{}
		fig.LoadAllConfigs("config.yml", &tmp)
		kingpeon.RegisterDynamicCommandsWithRunner(runner, app, data.CustomCommands, jiracli.TemplateProcessor())
	}

	app.Terminate(func(status int) {
		for _, arg := range os.Args {
			if arg == "-h" || arg == "--help" || len(os.Args) == 1 {
				panic(jiracli.Exit{Code: 0})
			}
		}
		panic(jiracli.Exit{Code: 1})
	})

	// checking for default usage of `jira ISSUE-123` but need to allow
	// for global options first like: `jira --user mothra ISSUE-123`
	ctx, _ := app.ParseContext(os.Args[1:])
	if ctx != nil {
		if ctx.SelectedCommand == nil {
			next := ctx.Next()
			if next != nil {
				if ok, err := regexp.MatchString("^[A-Z]+-[0-9]+$", next.Value); err != nil {
					log.Errorf("Invalid Regex: %s", err)
				} else if ok {
					// insert "view" at i=1 (2nd position)
					os.Args = append(os.Args[:1], append([]string{"view"}, os.Args[1:]...)...)
				}
			}
		}
	}

	if _, err := app.Parse(os.Args[1:]); err != nil {
		if _, ok := err.(*jiracli.Error); ok {
			log.Errorf("%s", err)
			panic(jiracli.Exit{Code: 1})
		} else {
			ctx, _ := app.ParseContext(os.Args[1:])
			if ctx != nil {
				app.UsageForContext(ctx)
			}
			log.Errorf("Invalid Usage: %s", err)
			panic(jiracli.Exit{Code: 1})
		}
	}
}
