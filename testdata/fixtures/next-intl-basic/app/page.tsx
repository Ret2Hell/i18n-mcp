import { useTranslations } from 'next-intl'

export default function Page() {
  const t = useTranslations('home')
  return <h1>{t('title')}</h1>
}
