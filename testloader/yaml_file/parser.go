package yaml_file

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"text/template"

	"gopkg.in/yaml.v2"
)

func parseTestDefinitionFile(absPath string) ([]Test, error) {
	data, err := ioutil.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	var testDefinitions []TestDefinition

	// reading the test source file
	if err := yaml.Unmarshal(data, &testDefinitions); err != nil {
		return nil, err
	}

	var tests []Test

	for _, definition := range testDefinitions {
		if testCases, err := makeTestFromDefinition(definition); err != nil {
			return nil, err
		} else {
			tests = append(tests, testCases...)
		}
	}

	return tests, nil
}

func executeTmpl(tmpl *template.Template, args map[string]interface{}) (string, error) {
	buf := &bytes.Buffer{}

	if err := tmpl.Execute(buf, args); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Make tests from the given test definition.
func makeTestFromDefinition(testDefinition TestDefinition) ([]Test, error) {
	var tests []Test

	// test definition has no cases, so using request/response as is
	if len(testDefinition.Cases) == 0 {
		test := Test{TestDefinition: testDefinition}
		test.Request = testDefinition.RequestTmpl
		test.Responses = testDefinition.ResponseTmpls
		test.BeforeScript = testDefinition.BeforeScriptParams.PathTmpl
		return append(tests, test), nil
	}

	requestTmpl, err := template.New("request").Parse(testDefinition.RequestTmpl)
	if err != nil {
		return nil, err
	}

	// produce as many tests as cases defined
	for caseIdx, testCase := range testDefinition.Cases {
		test := Test{TestDefinition: testDefinition}
		test.Name = fmt.Sprintf("%s #%d", test.Name, caseIdx)

		// compile request body
		test.Request, err = executeTmpl(requestTmpl, testCase.RequestArgs)

		// compile response bodies
		test.Responses = make(map[int]string)
		for status, tpl := range testDefinition.ResponseTmpls {
			args, ok := testCase.ResponseArgs[status]
			if ok {
				// found args for response status
				t, err := template.New("response").Parse(tpl)
				if err != nil {
					return nil, err
				}
				test.Responses[status], err = executeTmpl(t, args)
			} else {
				// not found args, using response as is
				test.Responses[status] = tpl
			}
		}

		// compile script body
		beforeScriptPathTmpl, err := template.New("beforeScript").Parse(testDefinition.BeforeScriptParams.PathTmpl)
		if err != nil {
			return nil, err
		}
		test.BeforeScript, err = executeTmpl(beforeScriptPathTmpl, testCase.BeforeScriptArgs)

		tests = append(tests, test)
	}

	return tests, nil
}
