package server

type SubscriptionAlreadyStreamingError struct{}

func (m *SubscriptionAlreadyStreamingError) Error() string {
	return "could not start streaming, subscription is already active"
}

type SubscriptionDoesNotExistError struct{}

func (m *SubscriptionDoesNotExistError) Error() string {
	return "could not start streaming, subscription does not exist"
}
