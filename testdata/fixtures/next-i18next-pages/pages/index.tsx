import { useTranslation } from 'next-i18next'

export default function Page() {
  const { t } = useTranslation()
  return <h1>{t('headline')}</h1>
}
