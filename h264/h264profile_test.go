package h264

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsingInvalid(t *testing.T) {
	// Malformed strings.
	assert.Nil(t, ParseProfileLevelId(""))
	assert.Nil(t, ParseProfileLevelId(" 42e01f"))
	assert.Nil(t, ParseProfileLevelId("4242e01f"))
	assert.Nil(t, ParseProfileLevelId("e01f"))
	assert.Nil(t, ParseProfileLevelId("gggggg"))

	// Invalid level.
	assert.Nil(t, ParseProfileLevelId("42e000"))
	assert.Nil(t, ParseProfileLevelId("42e00f"))
	assert.Nil(t, ParseProfileLevelId("42e0ff"))

	// Invalid profile.
	assert.Nil(t, ParseProfileLevelId("42e11f"))
	assert.Nil(t, ParseProfileLevelId("58601f"))
	assert.Nil(t, ParseProfileLevelId("64e01f"))
}

func TestParsingLevel(t *testing.T) {
	assert.Equal(t, ParseProfileLevelId("42e01f").Level, byte(Level3_1))
	assert.Equal(t, ParseProfileLevelId("42e00b").Level, byte(Level1_1))
	assert.Equal(t, ParseProfileLevelId("42f00b").Level, byte(Level1_b))
	assert.Equal(t, ParseProfileLevelId("42C02A").Level, byte(Level4_2))
	assert.Equal(t, ParseProfileLevelId("640c34").Level, byte(Level5_2))
}

func TestParsingConstrainedBaseline(t *testing.T) {
	assert.Equal(t, ParseProfileLevelId("42e01f").Profile, byte(ProfileConstrainedBaseline))
	assert.Equal(t, ParseProfileLevelId("42C02A").Profile, byte(ProfileConstrainedBaseline))
	assert.Equal(t, ParseProfileLevelId("4de01f").Profile, byte(ProfileConstrainedBaseline))
	assert.Equal(t, ParseProfileLevelId("58f01f").Profile, byte(ProfileConstrainedBaseline))
}

func TestParsingBaseline(t *testing.T) {
	assert.Equal(t, ParseProfileLevelId("42a01f").Profile, byte(ProfileBaseline))
	assert.Equal(t, ParseProfileLevelId("58A01F").Profile, byte(ProfileBaseline))
}

func TestParsingMain(t *testing.T) {
	assert.Equal(t, ParseProfileLevelId("4D401f").Profile, byte(ProfileMain))
}

func TestParsingHigh(t *testing.T) {
	assert.Equal(t, ParseProfileLevelId("64001f").Profile, byte(ProfileHigh))
}

func TestParsingConstrainedHigh(t *testing.T) {
	assert.Equal(t, ParseProfileLevelId("640c1f").Profile, byte(ProfileConstrainedHigh))
}

func TestToString(t *testing.T) {
	assert.Equal(t, ProfileLevelId{ProfileConstrainedBaseline, Level3_1}.String(), "42e01f")
	assert.Equal(t, ProfileLevelId{ProfileBaseline, Level1}.String(), "42000a")
	assert.Equal(t, ProfileLevelId{ProfileMain, Level3_1}.String(), "4d001f")
	assert.Equal(t, ProfileLevelId{ProfileConstrainedHigh, Level4_2}.String(), "640c2a")
	assert.Equal(t, ProfileLevelId{ProfileHigh, Level4_2}.String(), "64002a")
}

func TestToStringLevel1b(t *testing.T) {
	assert.Equal(t, ProfileLevelId{ProfileConstrainedBaseline, Level1_b}.String(), "42f00b")
	assert.Equal(t, ProfileLevelId{ProfileBaseline, Level1_b}.String(), "42100b")
	assert.Equal(t, ProfileLevelId{ProfileMain, Level1_b}.String(), "4d100b")
}

func TestParseToString(t *testing.T) {
	assert.Equal(t, ParseProfileLevelId("42e01f").String(), "42e01f")
	assert.Equal(t, ParseProfileLevelId("42E01F").String(), "42e01f")
	assert.Equal(t, ParseProfileLevelId("4d100b").String(), "4d100b")
	assert.Equal(t, ParseProfileLevelId("4D100B").String(), "4d100b")
	assert.Equal(t, ParseProfileLevelId("640c2a").String(), "640c2a")
	assert.Equal(t, ParseProfileLevelId("640C2A").String(), "640c2a")
}

func TestToStringInvalid(t *testing.T) {
	assert.Empty(t, NewProfileLevelId(ProfileHigh, Level1_b).String())
	assert.Empty(t, NewProfileLevelId(ProfileConstrainedHigh, Level1_b).String())
	assert.Empty(t, NewProfileLevelId(255, Level3_1).String())
}

func TestParseSdpProfileLevelIdEmpty(t *testing.T) {
	profileLevelId := ParseSdpProfileLevelId("")

	assert.NotNil(t, profileLevelId)
	assert.Equal(t, profileLevelId.Profile, byte(ProfileConstrainedBaseline))
	assert.Equal(t, profileLevelId.Level, byte(Level3_1))
}

func TestParseSdpProfileLevelIdConstrainedHigh(t *testing.T) {
	profileLevelId := ParseSdpProfileLevelId("640c2a")

	assert.NotNil(t, profileLevelId)
	assert.Equal(t, profileLevelId.Profile, byte(ProfileConstrainedHigh))
	assert.Equal(t, profileLevelId.Level, byte(Level4_2))
}

func TestParseSdpProfileLevelIdInvalid(t *testing.T) {
	assert.Nil(t, ParseSdpProfileLevelId("foobar"))
}

func TestIsSameProfile(t *testing.T) {
	assert.True(t, IsSameProfile("", ""))
	assert.True(t, IsSameProfile("42e01f", "42C02A"))
	assert.True(t, IsSameProfile("42a01f", "58A01F"))
	assert.True(t, IsSameProfile("42e01f", ""))
}

func TestIsNotSameProfile(t *testing.T) {
	assert.False(t, IsSameProfile("", "4d001f"))
	assert.False(t, IsSameProfile("42a01f", "640c1f"))
	assert.False(t, IsSameProfile("42000a", "64002a"))
}

func TestGenerateProfileLevelIdForAnswerEmpty(t *testing.T) {
	answer, _ := GenerateProfileLevelIdForAnswer(RtpParameter{}, RtpParameter{})

	assert.Empty(t, answer)
}

func TestGenerateProfileLevelIdForAnswerLevelSymmetryCapped(t *testing.T) {
	lowLevel := RtpParameter{ProfileLevelId: "42e015"}
	highLevel := RtpParameter{ProfileLevelId: "42e01f"}

	answer1, _ := GenerateProfileLevelIdForAnswer(lowLevel, highLevel)
	answer2, _ := GenerateProfileLevelIdForAnswer(highLevel, lowLevel)

	assert.Equal(t, answer1, "42e015")
	assert.Equal(t, answer2, "42e015")
}

func TestGenerateProfileLevelIdForAnswerConstrainedBaselineLevelAsymmetry(t *testing.T) {
	localParams := RtpParameter{ProfileLevelId: "42e01f", LevelAsymmetryAllowed: 1}
	remoteParams := RtpParameter{ProfileLevelId: "42e015", LevelAsymmetryAllowed: 1}

	answer, _ := GenerateProfileLevelIdForAnswer(localParams, remoteParams)

	assert.Equal(t, answer, "42e01f")
}

func Test_byteMaskString(t *testing.T) {
	type args struct {
		c   byte
		str string
	}
	tests := []struct {
		name     string
		args     args
		wantMask byte
	}{
		{
			name:     "test1",
			args:     args{c: 'x', str: "x1xx0000"},
			wantMask: bitstoByte("10110000"),
		},
		{
			name:     "test2",
			args:     args{c: '1', str: "x1xx001x"},
			wantMask: bitstoByte("01000010"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotMask := byteMaskString(tt.args.c, tt.args.str); gotMask != tt.wantMask {
				t.Errorf("byteMaskString() = %v, want %v", gotMask, tt.wantMask)
			}
		})
	}
}

func bitstoByte(str string) byte {
	v, _ := strconv.ParseUint(str, 2, 32)
	return byte(v)
}
