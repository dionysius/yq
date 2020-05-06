package wrap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"

	"github.com/go-log/log"
	"gopkg.in/yaml.v3"
)

// Wrapper is responsible to execute the jq wrapping. Trying to handle, include and exclude arguments where needed accordingly. Translates the format from YAML to JSON in the input, and from JSON to YAML in the output.
type Wrapper struct {
	// Path to JQ, defaults to "jq" if left empty.
	JQ string
	// Args for JQ, exclude the script in the first element. You probably want to use os.Args[1:].
	Args []string
	// Pipe for Stdin, where input is read from, defaults to os.Stdin
	Stdin io.Reader
	// Pipe for Stdout, where output is written to, defaults to os.Stdout
	Stdout io.WriteCloser
	// Pipe for Stderr, where error is written to, defaults to os.Stderr
	Stderr io.WriteCloser
	// Define debug logger, which all steps are logged to if defined
	Debug log.Logger
	// ProcessState is available once Wrapper is Run()
	ProcessState *os.ProcessState

	wrappedArgs map[string]string
	noInArgs    map[string]string
	noOutArgs   map[string]string
	stdinForJQ  io.Reader
	stdoutForJQ *bytes.Buffer
}

// wrappedParams contains parameters for jq, which yq does not forward and sometimes applies it's own processing. Usually output format related, which are reimplemented to match the jq API. Possibly incomplete! Format, opt: longopt.
var wrappedParams = map[string]string{
	"-r": "--raw-output",
	"-R": "--raw-input",
}

// noInputParams contains parameters for jq, which yq must not translate and wait on stdin. Possibly incomplete! Format, opt: longopt.
var noInputParams = map[string]string{
	"-h": "--help",
}

// noOutputParams contains parameters for jq, which yq must not translate stdout. Possibly incomplete! Format, opt: longopt.
var noOutputParams = map[string]string{
	"-h": "--help",
}

// Run the wrapper and execute jq
func (w *Wrapper) Run() error {
	w.defaults()
	w.checkParams()

	err := w.processInput()
	if err != nil {
		return err
	}

	err = w.execute()
	if err != nil {
		return err
	}

	err = w.processOutput()

	return err
}

// Apply defaults where needed
func (w *Wrapper) defaults() {
	if w.JQ == "" {
		w.JQ = "jq"
	}

	if w.Stdin == nil {
		w.Stdin = os.Stdin
	}

	if w.Stdout == nil {
		w.Stdout = os.Stdout
	}

	if w.Stderr == nil {
		w.Stderr = os.Stderr
	}
}

// Look for specific arguments which need to be handled, discarded or excluded
func (w *Wrapper) checkParams() {
	w.wrappedArgs = map[string]string{}

	// wrapped params are cut out from jq args, but remembered
	for k, v := range wrappedParams {
		i := 0

		for _, a := range w.Args {
			switch a {
			case k:
				fallthrough
			case v:
				w.wrappedArgs[k] = strconv.FormatBool(true)
			default:
				w.Args[i] = a
				i++
			}
		}

		w.Args = w.Args[:i]
	}

	w.debug("wrappedArgs", w.wrappedArgs)

	w.noInArgs = map[string]string{}

	// noout params indicate no input translation must be done
	for k, v := range noInputParams {
		for _, a := range w.Args {
			switch a {
			case k:
				fallthrough
			case v:
				w.noInArgs[k] = strconv.FormatBool(true)
			}
		}
	}

	w.debug("noInArgs", w.noInArgs)

	w.noOutArgs = map[string]string{}

	// noout params indicate no output translation must be done
	for k, v := range noOutputParams {
		for _, a := range w.Args {
			switch a {
			case k:
				fallthrough
			case v:
				w.noOutArgs[k] = strconv.FormatBool(true)
			}
		}
	}

	w.debug("noOutArgs", w.noOutArgs)
}

// Output the debug info
func (w *Wrapper) debug(name string, data interface{}) {
	if w.Debug != nil {
		w.Debug.Logf("%s %T:%+q", name, data, data)
	}
}

// Process input for use with JQ
func (w *Wrapper) processInput() error {
	// if there a parameters which don't allow translation and waiting for stdin
	if len(w.noInArgs) > 0 {
		w.stdinForJQ = nil
		return nil
	}

	// get all data from stdin
	inYAML, err := ioutil.ReadAll(w.Stdin)
	if err != nil {
		return err
	}

	w.debug("inYAML", inYAML)

	// if stdin (= inYAML) is empty, no need to translate anything
	inJSON := []byte{}

	if len(inYAML) > 0 {
		// try to parse that yaml
		var inRAW interface{}

		// if w.wrappedArgs["-R"] != "" {
		// 	inRAW = inYAML
		// } else {
		err = yaml.Unmarshal(inYAML, &inRAW)
		if err != nil {
			return err
		}
		// }

		w.debug("inRAW", inRAW)

		// put that into json
		inJSON, err = json.Marshal(inRAW)
		if err != nil {
			return err
		}
	}

	w.debug("inJSON", inJSON)

	// setup reader for jq's stdin
	w.stdinForJQ = bytes.NewBuffer(inJSON)

	return nil
}

// Execute JQ and handle
func (w *Wrapper) execute() error {
	// setup output writers
	w.stdoutForJQ = bytes.NewBuffer([]byte{})

	w.debug("jqArgs", w.Args)

	// set up cmd
	cmd := exec.Command("jq", w.Args...) // nolint: gosec
	cmd.Stdin = w.stdinForJQ
	cmd.Stdout = w.stdoutForJQ
	cmd.Stderr = w.Stderr

	err := cmd.Run()
	if err != nil {
		// an exit error from the underlying command is an expected response
		if _, is := err.(*exec.ExitError); !is {
			return err
		}
	}

	w.debug("cmdErr", err)

	w.ProcessState = cmd.ProcessState

	return nil
}

// Process output
func (w *Wrapper) processOutput() error {
	// get all data from stdout
	outJSON := w.stdoutForJQ.Bytes()

	w.debug("outJSON", outJSON)

	// if stdout (= inJSON) is empty, no need to translate anything
	if len(outJSON) == 0 {
		return nil
	}

	// if there are no output translation arguments, just output as is
	if len(w.noOutArgs) > 0 {

		_, err := w.Stdout.Write(outJSON)

		return err
	}

	outYAML := []byte{}

	// try to parse that json
	var outRAW interface{}

	err := json.Unmarshal(outJSON, &outRAW)
	if err != nil {
		return err
	}

	w.debug("outRAW", outRAW)

	if w.wrappedArgs["-r"] != "" {
		outYAML = []byte(fmt.Sprintf("%v", outRAW))
	} else {
		// and put that to yaml
		outYAML, err = yaml.Marshal(outRAW)
		if err != nil {
			return err
		}
	}

	w.debug("outYAML", outYAML)

	_, err = w.Stdout.Write(outYAML)

	return err
}
