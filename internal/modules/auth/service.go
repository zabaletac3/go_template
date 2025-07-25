

type AuthService struct {
	repo repositories.UserRepositoryInterface
	logger interfaces.LoggerInterface
}

func NewAuthService(
	repo repositories.UserRepositoryInterface, 
	logger interfaces.LoggerInterface) *AuthService {
	return &AuthService{repo: repo, logger: logger.With("service", "auth")}
}


func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest) (*models.User, error) {
	s.logger.Info("Login request received", "email", req.Email)

	
}