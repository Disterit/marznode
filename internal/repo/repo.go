package repo

type MarznodeRepo interface {
}

type Repository struct {
	MarznodeRepo MarznodeRepo
}

func NewRepository(marznodeRepo MarznodeRepo) *Repository {
	return &Repository{
		MarznodeRepo: marznodeRepo,
	}
}
