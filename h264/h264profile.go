package h264

import (
	"errors"
	"fmt"
	"math"
	"strconv"
)

const (
	ProfileConstrainedBaseline byte = 1
	ProfileBaseline                 = 2
	ProfileMain                     = 3
	ProfileConstrainedHigh          = 4
	ProfileHigh                     = 5

	// All values are equal to ten times the level number, except level 1b which is
	// special.
	Level1_b byte = 0
	Level1        = 10
	Level1_1      = 11
	Level1_2      = 12
	Level1_3      = 13
	Level2        = 20
	Level2_1      = 21
	Level2_2      = 22
	Level3        = 30
	Level3_1      = 31
	Level3_2      = 32
	Level4        = 40
	Level4_1      = 41
	Level4_2      = 42
	Level5        = 50
	Level5_1      = 51
	Level5_2      = 52
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

/**
 * Returns canonical string representation as three hex bytes of the profile
 * level id, or returns nothing for invalid profile level ids.
 */
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

	default:
		return ""
	}

	return fmt.Sprintf("%s%02x", profileIdcIopString, profileLevelId.Level)
}

// Default ProfileLevelId.
//
// TODO: The default should really be profile Baseline and level 1 according to
// the spec: https://tools.ietf.org/html/rfc6184#section-8.1. In order to not
// break backwards compatibility with older versions of WebRTC where external
// codecs don"t have any parameters, use profile ConstrainedBaseline level 3_1
// instead. This workaround will only be done in an interim period to allow
// external clients to update their code.
//
// http://crbug/webrtc/6337.
var DefaultProfileLevelId = ProfileLevelId{
	Profile: ProfileConstrainedBaseline,
	Level:   Level3_1,
}

// For level_idc=11 and profile_idc=0x42, 0x4D, or 0x58, the constraint set3
// flag specifies if level 1b or level 1.1 is used.
const ConstraintSet3Flag byte = 0x10

// Class for matching bit patterns such as "x1xx0000" where "x" is allowed to be
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

// Class for converting between profile_idc/profile_iop to Profile.
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

// This is from https://tools.ietf.org/html/rfc6184#section-8.1.
var ProfilePatterns = []ProfilePattern{
	{0x42, NewBitPattern("x1xx0000"), ProfileConstrainedBaseline},
	{0x4D, NewBitPattern("1xxx0000"), ProfileConstrainedBaseline},
	{0x58, NewBitPattern("11xx0000"), ProfileConstrainedBaseline},
	{0x42, NewBitPattern("x0xx0000"), ProfileBaseline},
	{0x58, NewBitPattern("10xx0000"), ProfileBaseline},
	{0x4D, NewBitPattern("0x0x0000"), ProfileMain},
	{0x64, NewBitPattern("00000000"), ProfileHigh},
	{0x64, NewBitPattern("00001100"), ProfileConstrainedHigh},
}

/**
 * Parse profile level id that is represented as a string of 3 hex bytes.
 * Nothing will be returned if the string is not a recognized H264 profile
 * level id.
 *
 * @param str - profile-level-id value as a string of 3 hex bytes.
 */
func ParseProfileLevelId(str string) (profileLevelId *ProfileLevelId) {
	// The string should consist of 3 bytes in hexadecimal format.
	if len(str) != 6 {
		return
	}
	profileLevelIdNumeric, _ := strconv.ParseInt(str, 16, 32)
	if profileLevelIdNumeric == 0 {
		return
	}
	// Separate into three bytes.
	levelIdc := byte(profileLevelIdNumeric & 0xFF)
	profileIop := byte(profileLevelIdNumeric >> 8 & 0xFF)
	profileIdc := byte(profileLevelIdNumeric >> 16 & 0xFF)

	// Parse level based on level_idc and constraint set 3 flag.
	var level byte

	switch levelIdc {
	case Level1_1:
		if (profileIop & ConstraintSet3Flag) != 0 {
			level = Level1_b
		} else {
			level = Level1_1
		}
	case Level1, Level1_2, Level1_3, Level2, Level2_1, Level2_2,
		Level3, Level3_1, Level3_2, Level4, Level4_1, Level4_2,
		Level5, Level5_1, Level5_2:
		level = levelIdc
	default:
		return
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

	return
}

/**
 * Parse profile level id that is represented as a string of 3 hex bytes
 * A default profile level id will be returned if profile level id is empty.
 *
 */
func ParseSdpProfileLevelId(profileLevelIdStr string) *ProfileLevelId {
	if len(profileLevelIdStr) == 0 {
		return &DefaultProfileLevelId
	}
	return ParseProfileLevelId(profileLevelIdStr)
}

/**
 * Returns true if the parameters have the same H264 profile, i.e. the same
 * H264 profile (Baseline, High, etc).
 *
 */
func IsSameProfile(profileLevelIdStr1, profileLevelIdStr2 string) bool {
	profileLevelId1 := ParseSdpProfileLevelId(profileLevelIdStr1)
	profileLevelId2 := ParseSdpProfileLevelId(profileLevelIdStr2)

	return profileLevelId1 != nil && profileLevelId2 != nil &&
		profileLevelId1.Profile == profileLevelId2.Profile
}

type RtpParameter struct {
	PacketizationMode     int    `json:"packetization-mode,omitempty"`
	ProfileLevelId        string `json:"profile-level-id,omitempty"`
	LevelAsymmetryAllowed int    `json:"level-asymmetry-allowed,omitempty"`
}

/**
 * Generate codec parameters that will be used as answer in an SDP negotiation
 * based on local supported parameters and remote offered parameters. Both
 * local_supported_params and remote_offered_params represent sendrecv media
 * descriptions, i.e they are a mix of both encode and decode capabilities. In
 * theory, when the profile in local_supported_params represent a strict superset
 * of the profile in remote_offered_params, we could limit the profile in the
 * answer to the profile in remote_offered_params.
 *
 * However, to simplify the code, each supported H264 profile should be listed
 * explicitly in the list of local supported codecs, even if they are redundant.
 * Then each local codec in the list should be tested one at a time against the
 * remote codec, and only when the profiles are equal should this func be
 * called. Therefore, this func does not need to handle profile intersection,
 * and the profile of local_supported_params and remote_offered_params must be
 * equal before calling this func. The parameters that are used when
 * negotiating are the level part of profile-level-id and level-asymmetry-allowed.
 *
 * @returns Canonical string representation as three hex bytes of the
 *   profile level id, or null if no one of the params have profile-level-id.
 *
 */
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

// Convert a string of 8 characters into a byte where the positions containing
// character c will have their bit set. For example, c = "x", str = "x1xx0000"
// will return 0b10110000.
func byteMaskString(c byte, str string) (mask byte) {
	length := len(str)

	for i, b := range str {
		if c == byte(b) {
			mask |= 1 << uint(length-1-i)
		}
	}

	return
}

// Compare H264 levels and handle the level 1b case.
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
