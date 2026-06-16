import useTranslation from 'next-translate/useTranslation'

export default function Page() {
  const { t } = useTranslation('common')
  return <button>{t('checkout')}</button>
}
