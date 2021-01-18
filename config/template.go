package config

import (
	"bufio"
	"bytes"
	"strings"
	"text/template"

	"golang.org/x/xerrors"
)

func CompileTemplate(s string, vars *Vars) ([]byte, error) {
	var t *template.Template
	if i, err := template.New("s").Funcs(vars.FuncMap()).Parse(s); err != nil {
		return nil, err
	} else {
		t = i
	}

	var bf bytes.Buffer
	if err := t.Execute(&bf, vars.Map()); err != nil {
		return nil, err
	} else {
		sc := bufio.NewScanner(bytes.NewReader(bf.Bytes()))
		var ln int
		for sc.Scan() {
			l := sc.Text()
			if strings.Contains(l, "<no value>") {
				return nil, xerrors.Errorf("some variables are not replaced in template string, %q(line: %d)", l, ln)
			}
			ln++
		}

		if err := sc.Err(); err != nil {
			return nil, err
		}

		return bf.Bytes(), nil
	}
}
