package cli_test

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/tkellen/cli"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
)

func TestDispatch(t *testing.T) {
	table := map[string]struct {
		input       io.Reader
		expectedErr bool
	}{
		"bad config reader": {
			input:       ioutil.NopCloser(bytes.NewReader([]byte("invalidyaml"))),
			expectedErr: true,
		},
		"good config reader": {
			input:       ioutil.NopCloser(bytes.NewReader([]byte("targets:"))),
			expectedErr: false,
		},
	}
	for name, test := range table {
		t.Run(name, func(t *testing.T) {
			_, err := cli.NewConfigFile(test.input, map[string]string{})
			if test.expectedErr && err == nil {
				t.Fatal("expected error")
			}
			if !test.expectedErr && err != nil {
				t.Fatalf("did not expect error: %s", err)
			}
		})
	}

}

func TestConfigFile_String(t *testing.T) {
	expected := "targets:\n  test:\n    path: ~/app\n    type: localDisk\n"
	actual := &cli.ConfigFile{
		Targets: map[string]cli.ConfigTarget{
			"test": {
				"type": "localDisk",
				"path": "~/app",
			},
		},
	}
	if expected != fmt.Sprintf("%s", actual) {
		t.Fatalf("expected %s, got %s", expected, actual)
	}
}

func TestConfigFile_Create(t *testing.T) {
	existingTarget := cli.ConfigTarget{
		"key": "value",
	}
	table := map[string]struct {
		configFile     *cli.ConfigFile
		targetName     string
		storeType      string
		expectedTarget cli.ConfigTarget
	}{
		"new targets are added": {
			configFile: &cli.ConfigFile{
				Targets: map[string]cli.ConfigTarget{},
			},
			targetName:     "test",
			storeType:      "s3",
			expectedTarget: nil, // doesn't exist yet
		},
		"existing targets are not overwritten": {
			configFile: &cli.ConfigFile{
				Targets: map[string]cli.ConfigTarget{
					"existing": existingTarget,
				},
			},
			targetName:     "existing",
			storeType:      "s3",
			expectedTarget: existingTarget,
		},
	}
	for name, test := range table {
		t.Run(name, func(t *testing.T) {
			test.configFile.Create(test.targetName, test.storeType)
			target, ok := test.configFile.Targets[test.targetName]
			if !ok {
				t.Fatal("expected target to be created")
			}
			if test.expectedTarget != nil && !reflect.DeepEqual(target, test.expectedTarget) {
				t.Fatalf("expected target %v, got %v", test.expectedTarget, target)
			}
		})
	}
}

func TestConfigFile_Target(t *testing.T) {
	expectedTarget := &cli.ConfigTarget{}
	table := map[string]struct {
		configFile  *cli.ConfigFile
		lookup      string
		expected    *cli.ConfigTarget
		expectedErr bool
	}{
		"existing target requested": {
			configFile: &cli.ConfigFile{
				Targets: map[string]cli.ConfigTarget{
					"test": *expectedTarget,
				},
			},
			lookup:      "test",
			expected:    expectedTarget,
			expectedErr: false,
		},
		"missing target requested": {
			configFile: &cli.ConfigFile{
				Targets: map[string]cli.ConfigTarget{},
			},
			lookup:      "test",
			expected:    expectedTarget,
			expectedErr: true,
		},
	}
	for name, test := range table {
		t.Run(name, func(t *testing.T) {
			actual, err := test.configFile.Target(test.lookup)
			if !test.expectedErr && err != nil {
				t.Fatalf("did not expect error %s", err)
			}
			if test.expected == actual {
				t.Fatalf("expected target %s, got %s", test.expected, actual)
			}
		})
	}
}

func TestConfigFile_Delete(t *testing.T) {
	table := map[string]struct {
		configFile          *cli.ConfigFile
		targetToDelete      string
		expectedTargetCount int
	}{
		"delete existing target": {
			configFile: &cli.ConfigFile{
				Targets: map[string]cli.ConfigTarget{
					"test": {},
				},
			},
			targetToDelete: "test",
		},
		"delete non-existing target": {
			configFile: &cli.ConfigFile{
				Targets: map[string]cli.ConfigTarget{
					"nope": {},
				},
			},
			targetToDelete: "test",
		},
	}
	for name, test := range table {
		t.Run(name, func(t *testing.T) {
			test.configFile.Delete(test.targetToDelete)
			if _, ok := test.configFile.Targets[test.targetToDelete]; ok {
				t.Fatal("deleted target still present")
			}
		})
	}
}

func TestConfigFile_Load(t *testing.T) {
	goodInput := []byte("targets:\n  test:\n    path: ~/app\n    type: localDisk\n")
	table := map[string]struct {
		input       io.Reader
		expected    []byte
		expectedErr error
	}{
		"load valid yaml": {
			input:       bytes.NewReader(goodInput),
			expected:    goodInput,
			expectedErr: nil,
		},
		"load invalid yaml": {
			input:       bytes.NewReader([]byte("notyaml")),
			expected:    []byte("targets: {}\n"),
			expectedErr: errors.New("cannot unmarshal"),
		},
		"load bad reader": {
			input:       iotest.TimeoutReader(bytes.NewReader([]byte("notyaml"))),
			expected:    []byte("targets: {}\n"),
			expectedErr: errors.New("timeout"),
		},
	}
	for name, test := range table {
		t.Run(name, func(t *testing.T) {
			configFile := cli.ConfigFile{}
			err := configFile.Load(test.input)
			if test.expectedErr == nil && err != nil {
				t.Fatalf("did not expect error: %s", err)
			}
			if err != nil && test.expectedErr != nil && !strings.Contains(err.Error(), test.expectedErr.Error()) {
				t.Fatalf("expected error: %s, got %s", test.expectedErr, err)
			}
			actual, _ := yaml.Marshal(configFile)
			if !bytes.Equal(test.expected, actual) {
				t.Fatalf("load failed, expected %s, got %s", test.expected, actual)
			}
		})
	}
}

func TestConfigFile_Save(t *testing.T) {
	cfg := &cli.ConfigFile{
		Targets: map[string]cli.ConfigTarget{
			"test": {
				"type": "localDisk",
				"path": "~/app",
			},
		},
	}
	badReadWriter, err := ioutil.TempFile("", "*")
	if err != nil {
		t.Fatalf("setting up test: %s", err)
	}
	defer os.RemoveAll(badReadWriter.Name())
	badReadWriter.Close()
	table := map[string]struct {
		configFile   *cli.ConfigFile
		readerWriter io.ReadWriter
		expected     []byte
		expectedErr  error
	}{
		"success": {
			configFile:   cfg,
			readerWriter: bytes.NewBuffer([]byte{}),
			expected:     []byte("targets:\n  test:\n    path: ~/app\n    type: localDisk\n"),
			expectedErr:  nil,
		},
		"failure": {
			configFile:   cfg,
			readerWriter: badReadWriter,
			expected:     nil,
			expectedErr:  errors.New("already closed"),
		},
	}
	for name, test := range table {
		t.Run(name, func(t *testing.T) {
			err := test.configFile.Save(test.readerWriter)
			if test.expectedErr == nil && err != nil {
				t.Fatalf("did not expect error: %s", err)
			}
			if err != nil && test.expectedErr != nil && !strings.Contains(err.Error(), test.expectedErr.Error()) {
				t.Fatalf("expected error: %s, got %s", test.expectedErr, err)
			}
			actual, _ := ioutil.ReadAll(test.readerWriter)
			if !bytes.Equal(test.expected, actual) {
				t.Fatalf("save failed, expected %s, got %s", test.expected, actual)
			}
		})
	}
}

func TestTarget_Set(t *testing.T) {
	target := &cli.ConfigTarget{}
	target.Set("key", "value")
	if len(*target) != 1 {
		t.Fatal("expected one item in target configuration")
	}
}

func TestTarget_Get(t *testing.T) {
	expected := "value"
	target := &cli.ConfigTarget{"key": expected}
	actual := target.Get("key")
	if expected != actual {
		t.Fatalf("expected %s, got %s", expected, actual)
	}
}

func TestTarget_Delete(t *testing.T) {
	target := (&cli.ConfigTarget{"key": "value"}).Delete("key")
	if _, ok := (*target)["key"]; ok {
		t.Fatal("expected key to be removed.")
	}
}
