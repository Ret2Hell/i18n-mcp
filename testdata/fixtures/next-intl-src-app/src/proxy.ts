import createMiddleware from 'next-intl/middleware'

export default createMiddleware({
  locales: ['en-US', 'ja'],
  defaultLocale: 'en-US'
})
