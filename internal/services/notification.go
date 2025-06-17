package services

type NotificationService struct {
}

func NewNotificationService() *NotificationService {
	return &NotificationService{}
}

var ResponsiblesByDeviceId = map[int32][]string{}

func (ns *NotificationService) SetResponsiblesByDeviceId(resposiblesByDeviceID map[int32][]string) {
	ResponsiblesByDeviceId = resposiblesByDeviceID
}

func (ns *NotificationService) GetResposibles(deviceID int32) []string {
	return ResponsiblesByDeviceId[deviceID]
}
