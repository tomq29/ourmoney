package transaction

import "errors"

var ErrZeroAmount = errors.New("Сумма ≤ 0")
var ErrMissingAccount = errors.New("не указан обязательный счет")
var ErrSameAccount = errors.New("перевод денег на тот же счет")
var ErrMissingDate = errors.New("нет даты")
var ErrMissingUserID = errors.New("не указан ID пользователя")
