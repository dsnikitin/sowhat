package salute

import "github.com/dsnikitin/sowhat/internal/consts/format"

type Options struct {
	Model         string `json:"model"`
	AudioEncoding string `json:"audio_encoding"`
	Language      string `json:"language"`
	ChannelsCount int    `json:"channels_count"`
}

type Request struct {
	Options       Options `json:"options"`
	RequestFileID string  `json:"request_file_id"`
}

func NewRequstByAudioFormat(fileID string, ft format.Type) *Request {
	req := &Request{
		Options: Options{
			Model:    "general",
			Language: "ru-RU",
		},
		RequestFileID: fileID,
	}

	switch ft {
	case format.MP3:
		req.Options.AudioEncoding = "MP3"
		req.Options.ChannelsCount = 2
	case format.OPUS:
		req.Options.AudioEncoding = "OPUS"
		req.Options.ChannelsCount = 1
	}

	return req
}
