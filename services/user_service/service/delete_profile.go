package service

import "context"

func (s UserService) DeleteProfile(ctx context.Context, id string) error {
	return s.userRepo.DeleteProfile(ctx, id)
}
