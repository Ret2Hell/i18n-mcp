import { useTranslations } from 'next-intl'
import { LocaleSwitcher } from '@/components/locale-switcher'

export default function Page() {
  const t = useTranslations('home')
  return <main><h1>{t('title')}</h1><p>{t('subtitle')}</p><LocaleSwitcher /></main>
}
