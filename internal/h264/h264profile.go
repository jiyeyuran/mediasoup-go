package h264

import (
	"errors"
	"fmt"
	"math"
	"strconv"
)

const (
	ProfileConstrainedBaseline byte = 1
	ProfileBaseline            byte = 2
	ProfileMain                byte = 3
	ProfileConstrainedHigh     byte = 4
	ProfileHigh                byte = 5
	ProfilePredictiveHigh444   byte = 6

	// All values are equal to ten times the level number, except level 1b which is
	// special.
	Level1_b byte = 0
	Level1   byte = 10
	Level1_1 byte = 11
	Level1_2 byte = 12
	Level1_3 byte = 13
	Level2   byte = 20
	Level2_1 byte = 21
	Level2_2 byte = 22
	Level3   byte = 30
	Level3_1 byte = 31
	Level3_2 byte = 32
	Level4   byte = 40
	Level4_1 byte = 41
	Level4_2 byte = 42
	Level5   byte = 50
	Level5_1 byte = 51
	Level5_2 byte = 52
)

type ProfileLevelId struct {
	Profile byte
	Level   byte
}

func NewProfileLevelId(profile, level byte) ProfileLevelId {
	return ProfileLevelId{
		Profile: profile,
		Level:   level,
	}
}

// String returns canonical string representation as three hex bytes of the profile
// level id, or returns nothing for invalid profile level ids.
func (profileLevelId ProfileLevelId) String() string {
	// Handle special case level == 1b.
	if profileLevelId.Level == Level1_b {
		switch profileLevelId.Profile {
		case ProfileConstrainedBaseline:
			return "42f00b"
		case ProfileBaseline:
			return "42100b"
		case ProfileMain:
			return "4d100b"
		default:
			return ""
		}
	}

	var profileIdcIopString string

	switch profileLevelId.Profile {
	case ProfileConstrainedBaseline:
		profileIdcIopString = "42e0"

	case ProfileBaseline:
		profileIdcIopString = "4200"

	case ProfileMain:
		profileIdcIopString = "4d00"

	case ProfileConstrainedHigh:
		profileIdcIopString = "640c"

	case ProfileHigh:
		profileIdcIopString = "6400"

	case ProfilePredictiveHigh444:
		profileIdcIopString = "f400"

	default:
		return ""
	}

	return fmt.Sprintf("%s%02x", profileIdcIopString, profileLevelId.Level)
}

// BitPattern is class for matching bit patterns such as "x1xx0000" where "x" is allowed to be
// either 0 or 1.
type BitPattern struct {
	mask        byte
	maskedValue byte
}

func NewBitPattern(str string) BitPattern {
	return BitPattern{
		mask:        math.MaxUint8 - byteMaskString('x', str),
		maskedValue: byteMaskString('1', str),
	}
}

func (b BitPattern) isMatch(value byte) bool {
	return b.maskedValue == (value & b.mask)
}

// ProfilePattern is class for converting between profile_idc/profile_iop to Profile.
type ProfilePattern struct {
	profileIdc byte
	profileIop BitPattern
	profile    byte
}

func NewProfilePattern(
	profileIdc byte,
	profileIop BitPattern,
	profile byte,
) ProfilePattern {
	return ProfilePattern{
		profileIdc: profileIdc,
		profileIop: profileIop,
		profile:    profile,
	}
}

// ProfilePatterns is from https://tools.ietf.org/html/rfc6184#section-8.1.
var ProfilePatterns = []ProfilePattern{
	{0x42, NewBitPattern("x1xx0000"), ProfileConstrainedBaseline},
	{0x4D, NewBitPattern("1xxx0000"), ProfileConstrainedBaseline},
	{0x58, NewBitPattern("11xx0000"), ProfileConstrainedBaseline},
	{0x42, NewBitPattern("x0xx0000"), ProfileBaseline},
	{0x58, NewBitPattern("10xx0000"), ProfileBaseline},
	{0x4D, NewBitPattern("0x0x0000"), ProfileMain},
	{0x64, NewBitPattern("00000000"), ProfileHigh},
	{0x64, NewBitPattern("00001100"), ProfileConstrainedHigh},
	{0xF4, NewBitPattern("00000000"), ProfilePredictiveHigh444},
}

type LevelConstraint struct {
	MaxMacroblocksPerSecond uint32
	MaxMacroblockFrameSize  uint32
	Level                   byte
}

// This is from ITU-T H.264 (02/2016) Table A-1 – Level limits.
var LevelConstraints = []LevelConstraint{
	{
		MaxMacroblocksPerSecond: 1485,
		MaxMacroblockFrameSize:  99,
		Level:                   Level1,
	},
	{
		MaxMacroblocksPerSecond: 1485,
		MaxMacroblockFrameSize:  99,
		Level:                   Level1_b,
	},
	{
		MaxMacroblocksPerSecond: 3000,
		MaxMacroblockFrameSize:  396,
		Level:                   Level1_1,
	},
	{
		MaxMacroblocksPerSecond: 6000,
		MaxMacroblockFrameSize:  396,
		Level:                   Level1_2,
	},
	{
		MaxMacroblocksPerSecond: 11880,
		MaxMacroblockFrameSize:  396,
		Level:                   Level1_3,
	},
	{
		MaxMacroblocksPerSecond: 11880,
		MaxMacroblockFrameSize:  396,
		Level:                   Level2,
	},
	{
		MaxMacroblocksPerSecond: 19800,
		MaxMacroblockFrameSize:  792,
		Level:                   Level2_1,
	},
	{
		MaxMacroblocksPerSecond: 20250,
		MaxMacroblockFrameSize:  1620,
		Level:                   Level2_2,
	},
	{
		MaxMacroblocksPerSecond: 40500,
		MaxMacroblockFrameSize:  1620,
		Level:                   Level3,
	},
	{
		MaxMacroblocksPerSecond: 108000,
		MaxMacroblockFrameSize:  3600,
		Level:                   Level3_1,
	},
	{
		MaxMacroblocksPerSecond: 216000,
		MaxMacroblockFrameSize:  5120,
		Level:                   Level3_2,
	},
	{
		MaxMacroblocksPerSecond: 245760,
		MaxMacroblockFrameSize:  8192,
		Level:                   Level4,
	},
	{
		MaxMacroblocksPerSecond: 245760,
		MaxMacroblockFrameSize:  8192,
		Level:                   Level4_1,
	},
	{
		MaxMacroblocksPerSecond: 522240,
		MaxMacroblockFrameSize:  8704,
		Level:                   Level4_2,
	},
	{
		MaxMacroblocksPerSecond: 589824,
		MaxMacroblockFrameSize:  22080,
		Level:                   Level5,
	},
	{
		MaxMacroblocksPerSecond: 983040,
		MaxMacroblockFrameSize:  36864,
		Level:                   Level5_1,
	},
	{
		MaxMacroblocksPerSecond: 2073600,
		MaxMacroblockFrameSize:  36864,
		Level:                   Level5_2,
	},
}

// ProfileToString prints name of given profile.
func ProfileToString(profile byte) string {
	switch profile {
	case ProfileConstrainedBaseline:
		return "ConstrainedBaseline"
	case ProfileBaseline:
		return "Baseline"
	case ProfileMain:
		return "Main"
	case ProfileConstrainedHigh:
		return "ConstrainedHigh"
	case ProfileHigh:
		return "High"
	default:
		return ""
	}
}

// LevelToString prints name of given level.
func LevelToString(level byte) string {
	switch level {
	case Level1_b:
		return "1b"
	case Level1:
		return "1"
	case Level1_1:
		return "1.1"
	case Level1_2:
		return "1.2"
	case Level1_3:
		return "1.3"
	case Level2:
		return "2"
	case Level2_1:
		return "2.1"
	case Level2_2:
		return "2.2"
	case Level3:
		return "3"
	case Level3_1:
		return "3.1"
	case Level3_2:
		return "3.2"
	case Level4:
		return "4"
	case Level4_1:
		return "4.1"
	case Level4_2:
		return "4.2"
	case Level5:
		return "5"
	case Level5_1:
		return "5.1"
	case Level5_2:
		return "5.2"
	default:
		return ""
	}
}

// ParseProfileLevelId parse profile level id that is represented as a string of 3 hex bytes.
// Nothing will be returned if the string is not a recognized H264 profile level id.
//
// @param str - profile-level-id value as a string of 3 hex bytes.
func ParseProfileLevelId(str string) (profileLevelId *ProfileLevelId) {
	// ConstraintSet3Flag for level_idc=11 and profile_idc=0x42, 0x4D, or 0x58, the constraint set3
	// flag specifies if level 1b or level 1.1 is used.
	const constraintSet3Flag byte = 0x10

	// The string should consist of 3 bytes in hexadecimal format.
	if len(str) != 6 {
		return nil
	}
	profileLevelIdNumeric, _ := strconv.ParseInt(str, 16, 32)
	if profileLevelIdNumeric == 0 {
		return nil
	}
	// Separate into three bytes.
	levelIdc := byte(profileLevelIdNumeric & 0xFF)
	profileIop := byte(profileLevelIdNumeric >> 8 & 0xFF)
	profileIdc := byte(profileLevelIdNumeric >> 16 & 0xFF)

	// Parse level based on level_idc and constraint set 3 flag.
	var level byte

	switch levelIdc {
	case Level1_1:
		if (profileIop & constraintSet3Flag) != 0 {
			level = Level1_b
		} else {
			level = Level1_1
		}
	case Level1, Level1_2, Level1_3, Level2, Level2_1, Level2_2,
		Level3, Level3_1, Level3_2, Level4, Level4_1, Level4_2,
		Level5, Level5_1, Level5_2:
		level = levelIdc
	default:
		return nil
	}

	// Parse profile_idc/profile_iop into a Profile enum.
	for _, pattern := range ProfilePatterns {
		if profileIdc == pattern.profileIdc &&
			pattern.profileIop.isMatch(profileIop) {
			return &ProfileLevelId{
				Profile: pattern.profile,
				Level:   level,
			}
		}
	}

	return nil
}

// ParseSdpProfileLevelId parse profile level id that is represented as a string of 3 hex bytes.
// A default profile level id will be returned if profile level id is empty.
func ParseSdpProfileLevelId(profileLevelIdStr string) *ProfileLevelId {
	// DefaultProfileLevelId.
	//
	// TODO: The default should really be profile Baseline and level 1 according to
	// the spec: https://tools.ietf.org/html/rfc6184#section-8.1. In order to not
	// break backwards compatibility with older versions of WebRTC where external
	// codecs don"t have any parameters, use profile ConstrainedBaseline level 3_1
	// instead. This workaround will only be done in an interim period to allow
	// external clients to update their code.
	//
	// http://crbug/webrtc/6337.
	defaultProfileLevelId := &ProfileLevelId{
		Profile: ProfileConstrainedBaseline,
		Level:   Level3_1,
	}
	if len(profileLevelIdStr) == 0 {
		return defaultProfileLevelId
	}
	return ParseProfileLevelId(profileLevelIdStr)
}

// IsSameProfile returns true if the parameters have the same H264 profile, i.e. the same
// H264 profile (Baseline, High, etc).
func IsSameProfile(profileLevelIdStr1, profileLevelIdStr2 string) bool {
	profileLevelId1 := ParseSdpProfileLevelId(profileLevelIdStr1)
	profileLevelId2 := ParseSdpProfileLevelId(profileLevelIdStr2)

	return profileLevelId1 != nil && profileLevelId2 != nil &&
		profileLevelId1.Profile == profileLevelId2.Profile
}

// IsSameProfileAndLevel returns true if the codec parameters have the same H264 profile, i.e. the
// same H264 profile (Baseline, High, etc) and same level.
func IsSameProfileAndLevel(profileLevelIdStr1, profileLevelIdStr2 string) bool {
	profileLevelId1 := ParseSdpProfileLevelId(profileLevelIdStr1)
	profileLevelId2 := ParseSdpProfileLevelId(profileLevelIdStr2)

	return profileLevelId1 != nil && profileLevelId2 != nil &&
		profileLevelId1.Profile == profileLevelId2.Profile &&
		profileLevelId1.Level == profileLevelId2.Level
}

type RtpParameter struct {
	PacketizationMode     uint32 `json:"packetization-mode,omitempty"`
	ProfileLevelId        string `json:"profile-level-id,omitempty"`
	LevelAsymmetryAllowed uint32 `json:"level-asymmetry-allowed,omitempty"`
}

// GenerateProfileLevelIdForAnswer generate codec parameters that will be used as answer in an SDP
// negotiation based on local supported parameters and remote offered parameters. Both
// local_supported_params and remote_offered_params represent sendrecv media descriptions, i.e they
// are a mix of both encode and decode capabilities. In theory, when the profile in
// local_supported_params represent a strict superset of the profile in remote_offered_params, we
// could limit the profile in the answer to the profile in remote_offered_params.
//
// However, to simplify the code, each supported H264 profile should be listed
// explicitly in the list of local supported codecs, even if they are redundant.
// Then each local codec in the list should be tested one at a time against the
// remote codec, and only when the profiles are equal should this func be
// called. Therefore, this func does not need to handle profile intersection,
// and the profile of local_supported_params and remote_offered_params must be
// equal before calling this func. The parameters that are used when
// negotiating are the level part of profile-level-id and level-asymmetry-allowed.
//
// @returns Canonical string representation as three hex bytes of the
// profile level id, or null if no one of the params have profile-level-id.
func GenerateProfileLevelIdForAnswer(
	localSupportedParams,
	remoteOfferedParams RtpParameter,
) (str string, err error) {
	if len(localSupportedParams.ProfileLevelId) == 0 &&
		len(remoteOfferedParams.ProfileLevelId) == 0 {
		return "", nil
	}

	localProfileLevelId := ParseSdpProfileLevelId(localSupportedParams.ProfileLevelId)
	remoteProfileLevelId := ParseSdpProfileLevelId(remoteOfferedParams.ProfileLevelId)

	if localProfileLevelId == nil {
		err = errors.New("invalid local_profile_level_id")
		return
	}
	if remoteProfileLevelId == nil {
		err = errors.New("invalid remote_profile_level_id")
		return
	}
	if localProfileLevelId.Profile != remoteProfileLevelId.Profile {
		err = errors.New("H264 Profile mismatch")
		return
	}

	// Parse level information.
	levelAsymmetryAllowed :=
		localSupportedParams.LevelAsymmetryAllowed > 0 &&
			remoteOfferedParams.LevelAsymmetryAllowed > 0
	localLevel := localProfileLevelId.Level
	remoteLevel := remoteProfileLevelId.Level
	minLevel := minLevel(localLevel, remoteLevel)

	// Determine answer level. When level asymmetry is not allowed, level upgrade
	// is not allowed, i.e., the level in the answer must be equal to or lower
	// than the level in the offer.
	answerLevel := minLevel
	if levelAsymmetryAllowed {
		answerLevel = localLevel
	}

	// Return the resulting profile-level-id for the answer parameters.
	profileLevelId := ProfileLevelId{
		Profile: localProfileLevelId.Profile,
		Level:   answerLevel,
	}

	return profileLevelId.String(), nil
}

// SupportedLevel given that a decoder supports up to a given frame size (in pixels) at up to
// a given number of frames per second, return the highest H264 level where it
// can guarantee that it will be able to support all valid encoded streams that
// are within that level.
func SupportedLevel(maxFramePixelCount, maxFps uint32) (level byte, ok bool) {
	const PixelsPerMacroblock = 16 * 16

	for i := len(LevelConstraints) - 1; i >= 0; i-- {
		levelConstraint := LevelConstraints[i]

		if levelConstraint.MaxMacroblockFrameSize*PixelsPerMacroblock <= maxFramePixelCount &&
			levelConstraint.MaxMacroblocksPerSecond <= maxFps*levelConstraint.MaxMacroblockFrameSize {
			return levelConstraint.Level, true
		}
	}

	return 0, false
}

// byteMaskString convert a string of 8 characters into a byte where the positions containing
// character c will have their bit set. For example, c = "x", str = "x1xx0000" will return
// 0b10110000.
func byteMaskString(c byte, str string) (mask byte) {
	length := len(str)

	for i, b := range str {
		if c == byte(b) {
			mask |= 1 << uint(length-1-i)
		}
	}

	return
}

// isLessLevel compare H264 levels and handle the level 1b case.
func isLessLevel(a, b byte) bool {
	if a == Level1_b {
		return b != Level1 && b != Level1_b
	}

	if b == Level1_b {
		return a != Level1
	}

	return a < b
}

func minLevel(a, b byte) byte {
	if isLessLevel(a, b) {
		return a
	}
	return b
}
