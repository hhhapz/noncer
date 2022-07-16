package announcements

import (
	"reflect"
	"testing"
)

func Test_formatAnnouncement(t *testing.T) {
	type args struct {
		maxLen  int
		subject string
		body    string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "no-cutoff",
			args: args{
				maxLen:  100,
				subject: "test",
				body:    "test",
			},
			want: []string{"test"},
		},
		{
			name: "cutoff",
			args: args{
				maxLen:  10,
				subject: "test",
				body:    "0.1.2.3.4.5.6.7.8.9",
			},
			want: []string{"0.1.2.", "3.4.5.6.7.", "8.9"},
		},
		{
			name: "cutoff that ends in a dot",
			args: args{
				maxLen:  11,
				subject: "test",
				body:    ".1.2.3.4.5.6.7.",
			},
			want: []string{".1.2.3.", "4.5.6.7."},
		},
		{
			name: "we cant split it",
			args: args{
				maxLen:  10,
				subject: "test",
				body:    "0123456789",
			},
			want: []string{"012345", "6789"},
		},
		{
			name: "with end delimiter",
			args: args{
				maxLen:  25,
				subject: "test",
				body: `Full body test.
example announcement.
Insert third line.`,
			},
			want: []string{"Full body test.\n", "example announcement.\n", "Insert third line."},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildContents(tt.args.maxLen, tt.args.subject, tt.args.body); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\ngot:\n\t%q\nwant:\n\t%q", got, tt.want)
			}
		})
	}
}
