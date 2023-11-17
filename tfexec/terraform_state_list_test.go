package tfexec

import (
	"context"
	"reflect"
	"regexp"
	"sort"
	"testing"
)

func TestTerraformCLIStateList(t *testing.T) {
	state := NewState([]byte("dummy state"))
	stdout := `null_resource.bar
null_resource.foo
`

	cases := []struct {
		desc         string
		mockCommands []*mockCommand
		state        *State
		addresses    []string
		opts         []string
		want         []string
		ok           bool
	}{
		{
			desc: "no addresses and no opts",
			mockCommands: []*mockCommand{
				{
					args:     []string{"terraform", "state", "list"},
					stdout:   stdout,
					exitCode: 0,
				},
			},
			state: nil,
			want:  []string{"null_resource.bar", "null_resource.foo"},
			ok:    true,
		},
		{
			desc: "failed to run terraform state list",
			mockCommands: []*mockCommand{
				{
					args:     []string{"terraform", "state", "list"},
					exitCode: 1,
				},
			},
			state: nil,
			want:  nil,
			ok:    false,
		},
		{
			desc: "with addresses",
			mockCommands: []*mockCommand{
				{
					args:     []string{"terraform", "state", "list", "aws_instance.example", "module.example"},
					stdout:   stdout,
					exitCode: 0,
				},
			},
			state:     nil,
			addresses: []string{"aws_instance.example", "module.example"},
			want:      []string{"null_resource.bar", "null_resource.foo"},
			ok:        true,
		},
		{
			desc: "with opts",
			mockCommands: []*mockCommand{
				{
					args:     []string{"terraform", "state", "list", "-state=foo.tfstate", "-id=bar"},
					stdout:   stdout,
					exitCode: 0,
				},
			},
			state: nil,
			opts:  []string{"-state=foo.tfstate", "-id=bar"},
			want:  []string{"null_resource.bar", "null_resource.foo"},
			ok:    true,
		},
		{
			desc: "with addresses and opts",
			mockCommands: []*mockCommand{
				{
					args:     []string{"terraform", "state", "list", "-state=foo.tfstate", "-id=bar", "aws_instance.example", "module.example"},
					stdout:   stdout,
					exitCode: 0,
				},
			},
			state:     nil,
			addresses: []string{"aws_instance.example", "module.example"},
			opts:      []string{"-state=foo.tfstate", "-id=bar"},
			want:      []string{"null_resource.bar", "null_resource.foo"},
			ok:        true,
		},
		{
			desc: "with state",
			mockCommands: []*mockCommand{
				{
					args:     []string{"terraform", "state", "list", "-state=/path/to/tempfile", "-id=bar", "aws_instance.example", "module.example"},
					argsRe:   regexp.MustCompile(`^terraform state list -state=.+ -id=bar aws_instance.example module.example$`),
					stdout:   stdout,
					exitCode: 0,
				},
			},
			state:     state,
			addresses: []string{"aws_instance.example", "module.example"},
			opts:      []string{"-id=bar"},
			want:      []string{"null_resource.bar", "null_resource.foo"},
			ok:        true,
		},
		{
			desc: "with state and -state= (conflict error)",
			mockCommands: []*mockCommand{
				{
					args:     []string{"terraform", "state", "list", "-state=/path/to/tempfile", "-id=bar", "-state=foo.tfstate", "aws_instance.example", "module.example"},
					argsRe:   regexp.MustCompile(`^terraform state list -state=\S+ -id=bar -state=foo.tfstate aws_instance.example module.example$`),
					exitCode: 0,
				},
			},
			state:     state,
			addresses: nil,
			opts:      []string{"-id=bar", "-state=foo.tfstate"},
			want:      nil,
			ok:        false,
		},
		{
			desc: "no resources",
			mockCommands: []*mockCommand{
				{
					args:     []string{"terraform", "state", "list"},
					stdout:   "",
					exitCode: 0,
				},
			},
			state: nil,
			want:  []string{},
			ok:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			e := NewMockExecutor(tc.mockCommands)
			terraformCLI := NewTerraformCLI(e)
			terraformCLI.SetExecPath("terraform")
			got, err := terraformCLI.StateList(context.Background(), tc.state, tc.addresses, tc.opts...)
			if tc.ok && err != nil {
				t.Fatalf("unexpected err: %s", err)
			}
			if !tc.ok && err == nil {
				t.Fatal("expected to return an error, but no error")
			}
			if tc.ok && !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got: %v, want: %v", got, tc.want)
			}
		})
	}
}

func TestAccTerraformCLIStateList(t *testing.T) {
	SkipUnlessAcceptanceTestEnabled(t)

	source := `
resource "null_resource" "foo" {}
resource "null_resource" "bar" {}
`
	e := SetupTestAcc(t, source)
	terraformCLI := NewTerraformCLI(e)

	err := terraformCLI.Init(context.Background(), "-input=false", "-no-color")
	if err != nil {
		t.Fatalf("failed to run terraform init: %s", err)
	}

	err = terraformCLI.Apply(context.Background(), nil, "-input=false", "-no-color", "-auto-approve")
	if err != nil {
		t.Fatalf("failed to run terraform apply: %s", err)
	}

	got, err := terraformCLI.StateList(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("failed to run terraform state list: %s", err)
	}

	want := []string{"null_resource.foo", "null_resource.bar"}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got: %v, want: %v", got, want)
	}
}
