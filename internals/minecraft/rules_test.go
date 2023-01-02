package minecraft

import "testing"

func TestRule_appliesFor(t *testing.T) {
	type fields struct {
		Action   string
		OS       OS
		Features map[string]bool
	}
	type args struct {
		os   string
		arch string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "allow empty",
			fields: fields{
				Action: "allow",
			},
			args: args{
				os:   "linux",
				arch: "x86",
			},
			want: true,
		},
		{
			name: "allow os",
			fields: fields{
				Action: "allow",
				OS: OS{
					Name: "linux",
				},
			},
			args: args{
				os:   "linux",
				arch: "x86",
			},
			want: true,
		},
		{
			name: "allow arch",
			fields: fields{
				Action: "allow",
				OS: OS{
					Arch: "x86",
				},
			},
			args: args{
				os:   "linux",
				arch: "x86",
			},
			want: true,
		},
		{
			name: "allow os arch",
			fields: fields{
				Action: "allow",
				OS: OS{
					Name: "linux",
					Arch: "x86",
				},
			},
			args: args{
				os:   "linux",
				arch: "x86",
			},
			want: true,
		},
		{
			name: "disallow empty",
			fields: fields{
				Action: "disallow",
			},
			args: args{
				os:   "linux",
				arch: "x86",
			},
			want: true,
		},
		{
			name: "disallow os",
			fields: fields{
				Action: "disallow",
				OS: OS{
					Name: "linux",
				},
			},
			args: args{
				os:   "linux",
				arch: "x86",
			},
			want: false,
		},
		{
			name: "disallow arch",
			fields: fields{
				Action: "disallow",
				OS: OS{
					Arch: "x86",
				},
			},
			args: args{
				os:   "linux",
				arch: "x86",
			},
			want: false,
		},
		{
			name: "disallow os arch",
			fields: fields{
				Action: "disallow",
				OS: OS{
					Name: "linux",
					Arch: "x86",
				},
			},
			args: args{
				os:   "linux",
				arch: "x86",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Rule{
				Action:   tt.fields.Action,
				OS:       tt.fields.OS,
				Features: tt.fields.Features,
			}
			if got := r.appliesFor(tt.args.os, tt.args.arch); got != tt.want {
				t.Errorf("Rule.appliesFor() = %v, want %v", got, tt.want)
			}
		})
	}
}
