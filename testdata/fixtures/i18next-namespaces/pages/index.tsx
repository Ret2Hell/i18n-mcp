import { useTranslation } from 'react-i18next'

export default function Page() {
  const { t } = useTranslation('common')
  return <button>{t('save')}</button>
}
