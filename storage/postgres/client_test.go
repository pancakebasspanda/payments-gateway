package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_maskCardNumber(t *testing.T) {
	type args struct {
		in string
		r  rune
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Successfully mask all but 1st 4 and last 4 numbers of card",
			args: args{
				in: "378282246310005",
				r:  'X',
			},
			want: "3782XXXXXXX0005",
		},
		{
			name: "card has less than 6 digits", // TODO need this in validation of the request
			args: args{
				in: "378005",
				r:  'X',
			},
			want: "378005",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, maskCardNumber(tt.args.in, tt.args.r))

		})
	}
}
