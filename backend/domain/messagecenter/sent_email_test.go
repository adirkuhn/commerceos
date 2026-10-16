package messagecenter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mc "github.com/adirkuhn/commerceos/domain/messagecenter"
)

func TestSentEmailFilter_Normalized(t *testing.T) {
	assert.Equal(t, mc.SentEmailFilter{Page: 1, PerPage: 50}, mc.SentEmailFilter{}.Normalized())
	assert.Equal(t, mc.SentEmailFilter{Page: 3, PerPage: 200}, mc.SentEmailFilter{Page: 3, PerPage: 1000}.Normalized())
	assert.Equal(t, mc.SentEmailFilter{Page: 1, PerPage: 10}, mc.SentEmailFilter{Page: -2, PerPage: 10}.Normalized())
}

func TestSentEmailStatusAndSource_Valid(t *testing.T) {
	for _, s := range []mc.SentEmailStatus{mc.SentEmailSent, mc.SentEmailFailed, mc.SentEmailSkipped} {
		assert.True(t, s.Valid(), s)
	}
	assert.False(t, mc.SentEmailStatus("bounced").Valid())
	for _, s := range []mc.SentEmailSource{mc.SentEmailSourceAutomation, mc.SentEmailSourceNotification, mc.SentEmailSourceSignInLink, mc.SentEmailSourceNewsletter, mc.SentEmailSourceSupplierIssues} {
		assert.True(t, s.Valid(), s)
	}
	assert.False(t, mc.SentEmailSource("fax").Valid())
}
