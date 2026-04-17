package format

type Type string

const (
	Unknown   Type = "UNKNOWN"
	PCM_S16LE Type = "PCM_S16LE"
	OPUS      Type = "OPUS"
	MP3       Type = "MP3"
	FLAC      Type = "FLAC"
	ALAW      Type = "ALAW"
	MULAW     Type = "MULAW"
)

func SaluteSpeechSupported() []Type {
	return []Type{PCM_S16LE, OPUS, MP3, FLAC, ALAW, MULAW}
}

func FromMIME(mime string) Type {
	switch mime {
	case "audio/x-pcm":
		return PCM_S16LE
	case "audio/ogg":
		return OPUS
	case "audio/mpeg":
		return MP3
	case "audio/flac":
		return FLAC
	case "audio/pcma":
		return ALAW
	case "audio/pcmu":
		return MULAW
	default:
		return Unknown
	}
}
