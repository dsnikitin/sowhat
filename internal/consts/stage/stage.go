package stage

type Name string

const (
	Upload          Name = "Upload"
	Recognize       Name = "Recognize"
	Poll            Name = "Poll"
	Download        Name = "Download"
	Summarize       Name = "Summarize"
	UploadToChatter Name = "UploadToChatter"
	Finalize        Name = "Finalize"
)
