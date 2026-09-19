package repo

import "context"

type AdminRepository interface {
	Delete(ctx context.Context, id uint) error
}
