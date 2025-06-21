package repo

type User interface {
	GetUserInfo() error        // list_users
	GetByInbound() error       // list_inbound_users
	RemoveUser() error         // remove_user
	UpdateUserInbounds() error // update_user_inbounds
	FlushUser() error          // flush_users
}

type Inbound interface {
	GetAllInbounds() error   // list_inbounds
	GetInboundsByTag() error // list_inbounds
	RegisterInbound() error  // register_inbound
	RemoveInbound() error    // remove_inbound
}

type Repository struct {
	User    User
	Inbound Inbound
}

func NewRepository(inbound Inbound, user User) *Repository {
	return &Repository{
		Inbound: inbound,
		User:    user,
	}
}
