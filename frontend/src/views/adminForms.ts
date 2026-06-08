export interface AddAdminForm {
  currentPass: string
  username: string
  password: string
  confirmPassword: string
}

export interface DeleteAdminForm {
  currentPass: string
}

export const normalizeAdminUsername = (username: string): string => username.trim()

export const isAddAdminFormComplete = (form: AddAdminForm): boolean => {
  return form.currentPass.length > 0 &&
    normalizeAdminUsername(form.username).length > 0 &&
    form.password.length > 0 &&
    form.confirmPassword.length > 0
}

export const addAdminPasswordsMatch = (form: AddAdminForm): boolean => {
  return form.password === form.confirmPassword
}

export const isAddAdminFormValid = (form: AddAdminForm): boolean => {
  return isAddAdminFormComplete(form) && addAdminPasswordsMatch(form)
}

export const isDeleteAdminFormValid = (form: DeleteAdminForm): boolean => {
  return form.currentPass.length > 0
}
