// =================================================================
//
// Work of the U.S. Department of Defense, Defense Digital Service.
// Released as open source under the MIT License.  See LICENSE file.
//
// =================================================================

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	JCVersion = "1.0.0"
)

func compress(input string, quotes int, last byte) ([]byte, int, byte) {
	output := make([]byte, 0)
	if len(input) == 0 {
		return output, quotes, last
	}
	if quotes == 0 {
		switch c := input[0]; c {
		case '"':
			quotes += 1
			output = append(output, c)
		case '\n', '\r', '\t', ' ':
		default:
			output = append(output, c)
		}
	} else {
		if input[0] == '"' && last != '\\' {
			quotes -= 1
		}
		output = append(output, input[0])
	}
	last = input[0]
	for i := 1; i < len(input); i++ {
		if quotes == 0 {
			switch c := input[i]; c {
			case '"':
				quotes += 1
				output = append(output, c)
			case '\n', '\r', '\t', ' ':
			default:
				output = append(output, c)
			}
		} else {
			if input[i] == '"' && last != '\\' {
				quotes -= 1
			}
			output = append(output, input[i])
		}
		last = input[i]
	}
	return output, quotes, last
}

func initFlags(flag *pflag.FlagSet) {
	flag.BoolP("version", "v", false, "show version")
}

func initViper(cmd *cobra.Command) (*viper.Viper, error) {
	v := viper.New()
	err := v.BindPFlags(cmd.Flags())
	if err != nil {
		return v, fmt.Errorf("error binding flag set to viper: %w", err)
	}
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv() // set environment variables to overwrite config
	return v, nil
}

func main() {

	rootCommand := cobra.Command{
		Use:                   `jc [flags]`,
		DisableFlagsInUseLine: true,
		Short:                 "jc is a simple tool for compressing JSON.",
		Long:                  "jc is a simple tool for compressing JSON.  jc does no input validation.  jc reads from stdin and writes to stdout.",
		SilenceErrors:         true,
		SilenceUsage:          true,
		RunE: func(cmd *cobra.Command, args []string) error {
			v, err := initViper(cmd)
			if err != nil {
				return fmt.Errorf("error initializing viper: %w", err)
			}

			if len(args) > 0 {
				return cmd.Usage()
			}

			if v.GetBool("version") {
				fmt.Println(JCVersion)
				return nil
			}

			stdin := os.Stdin
			stdout := os.Stdout

			reader := bufio.NewReader(stdin)
			writer := bufio.NewWriter(stdout)

			var wg sync.WaitGroup
			wg.Add(1)
			signals := make(chan os.Signal, 1)
			signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGPIPE)

			gracefulMutex := &sync.Mutex{}
			gracefulShutdown := false

			updateGracefulShutdown := func(value *bool) bool {
				gracefulMutex.Lock()
				if value != nil {
					gracefulShutdown = *value
				} else {
					value = &gracefulShutdown
				}
				gracefulMutex.Unlock()
				return *value
			}

			go func() {
				<-signals
				value := true
				updateGracefulShutdown(&value)
			}()

			brokenPipe := false
			quotes := 0
			last := byte(0)
			go func() {
				eof := false
				for (!updateGracefulShutdown(nil)) && (!eof) && (!brokenPipe) {

					b := make([]byte, 4096)
					n, errRead := reader.Read(b)
					if errRead != nil {
						if errRead == io.EOF {
							eof = true
						} else {
							_, _ = fmt.Fprintln(os.Stderr, fmt.Errorf("error reading from stdin: %w", errRead).Error())
							break
						}
					}

					if gracefulShutdown {
						break
					}

					if n > 0 {
						output, q, l := compress(string(b[:n]), quotes, last)
						_, errWrite := writer.Write(output)
						if errWrite != nil {
							if perr, ok := errWrite.(*os.PathError); ok {
								if perr.Err == syscall.EPIPE {
									brokenPipe = true
									break
								}
							}
							_, _ = fmt.Fprintln(os.Stderr, fmt.Errorf("error writing to stdout: %w", errWrite).Error())
						}
						quotes = q
						last = l
					}

				}
				wg.Done()
			}()

			wg.Wait() // wait until done writing or received signal for graceful shutdown

			if !brokenPipe {
				errFlush := writer.Flush()
				if errFlush != nil {
					_, _ = fmt.Fprintln(os.Stderr, fmt.Errorf("error flushing to stdout: %w", errFlush).Error())
				}
			}

			return nil
		},
	}
	initFlags(rootCommand.Flags())

	if err := rootCommand.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "jc: "+err.Error())
		_, _ = fmt.Fprintln(os.Stderr, "Try jc --help for more information.")
		os.Exit(1)
	}
}
