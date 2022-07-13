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
		want Announcement
	}{
		{
			name: "no-cutoff",
			args: args{
				maxLen:  100,
				subject: "test",
				body:    "test",
			},
			want: Announcement{
				Subject:  "test",
				Contents: []string{"test"},
			},
		},
		{
			name: "cutoff",
			args: args{
				maxLen:  10,
				subject: "test",
				body:    "0.1.2.3.4.5.6.7.8.9",
			},
			want: Announcement{
				Subject:  "test",
				Contents: []string{"0.1.2.", "3.4.5.6.7.", "8.9"},
			},
		},
		{
			name: "cutoff that ends in a dot",
			args: args{
				maxLen:  11,
				subject: "test",
				body:    ".1.2.3.4.5.6.7.",
			},
			want: Announcement{
				Subject:  "test",
				Contents: []string{".1.2.3.", "4.5.6.7."},
			},
		},
		{
			name: "we cant split it",
			args: args{
				maxLen:  10,
				subject: "test",
				body:    "0123456789",
			},
			want: Announcement{
				Subject:  "test",
				Contents: []string{"0123456789"},
			},
		},
		{
			name: "with end delimiter",
			args: args{
				maxLen:  10,
				subject: "test",
				body: `Full body test.
example announcement.
It has an end delimiter. \-\-
and then an email.
`,
			},
			want: Announcement{
				Subject:  "test",
				Contents: []string{"Full body test.", "example announcement.", "It has an end delimiter."},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatAnnouncement(tt.args.maxLen, tt.args.subject, tt.args.body); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("formatAnnouncement() = %q, want %q", got, tt.want)
			}
		})
	}
}
