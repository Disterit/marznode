package service

type Marznode interface {
}

type Service struct {
	MarzService Marznode
}

func NewService(marzService Marznode) *Service {
	return &Service{
		MarzService: marzService,
	}
}
