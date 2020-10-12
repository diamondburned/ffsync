package sync

type fileAction uint8

const (
	noAction fileAction = iota
	copyAction
	convertAction
)

type Options struct {
	FileFormats []string // to transcode
	CopyFormats []string // to copy
}

func (o Options) IsExt(ext string) bool {
	return o.action(ext) > noAction
}

func (o Options) action(ext string) fileAction {
	for _, f := range o.FileFormats {
		if f == ext {
			return convertAction
		}
	}
	for _, f := range o.CopyFormats {
		if f == ext {
			return copyAction
		}
	}
	return noAction
}
