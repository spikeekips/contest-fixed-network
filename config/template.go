package config

import (
	"bufio"
	"bytes"
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

func CompileTemplate(s string, vars *Vars) ([]byte, error) {
	t, err := template.New("s").Funcs(vars.FuncMap()).Parse(s)
	if err != nil {
		return nil, err
	}

	var bf bytes.Buffer
	if err := t.Execute(&bf, vars.Map()); err != nil {
		return nil, err
	}

	sc := bufio.NewScanner(bytes.NewReader(bf.Bytes()))
	var ln int
	for sc.Scan() {
		l := sc.Text()
		if strings.Contains(l, "<no value>") {
			return nil, errors.Errorf("some variables are not replaced in template string, %q(line: %d)", l, ln)
		}
		ln++
	}

	if err := sc.Err(); err != nil {
		return nil, err
	}

	return bf.Bytes(), nil
}
