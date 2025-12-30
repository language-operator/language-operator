/**
 * Currency formatting utilities
 */

interface CurrencyFormatOptions {
  locale?: string
  minimumFractionDigits?: number
  maximumFractionDigits?: number
}

/**
 * Formats a number as currency using Intl.NumberFormat
 */
export function formatCurrency(
  amount: number,
  currency: string = 'USD',
  options: CurrencyFormatOptions = {}
): string {
  const {
    locale = 'en-US',
    minimumFractionDigits = 2,
    maximumFractionDigits = 3
  } = options

  try {
    return new Intl.NumberFormat(locale, {
      style: 'currency',
      currency,
      minimumFractionDigits,
      maximumFractionDigits
    }).format(amount)
  } catch (error) {
    console.warn(`Failed to format currency ${currency}, falling back to USD:`, error)
    // Fallback to USD if the currency code is invalid
    return new Intl.NumberFormat(locale, {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits,
      maximumFractionDigits
    }).format(amount)
  }
}

/**
 * Formats currency with auto-precision based on amount
 * Small amounts (< 0.01) show up to 6 decimal places
 * Regular amounts show 2 decimal places
 */
export function formatCurrencyAutoPrecision(
  amount: number,
  currency: string = 'USD',
  locale: string = 'en-US'
): string {
  const absAmount = Math.abs(amount)
  
  if (absAmount < 0.01 && absAmount > 0) {
    // Show more decimal places for very small amounts
    return formatCurrency(amount, currency, {
      locale,
      minimumFractionDigits: 2,
      maximumFractionDigits: 6
    })
  }
  
  return formatCurrency(amount, currency, {
    locale,
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  })
}

/**
 * Currency mapping for common locale detection
 */
const LOCALE_CURRENCY_MAP: Record<string, string> = {
  'en-US': 'USD',
  'en-GB': 'GBP',
  'de-DE': 'EUR',
  'fr-FR': 'EUR',
  'es-ES': 'EUR',
  'it-IT': 'EUR',
  'ja-JP': 'JPY',
  'ko-KR': 'KRW',
  'zh-CN': 'CNY',
  'zh-TW': 'TWD'
}

/**
 * Get the user's preferred locale from browser
 */
export function getUserLocale(): string {
  if (typeof navigator !== 'undefined') {
    return navigator.language || 'en-US'
  }
  return 'en-US'
}

/**
 * Format currency using user's locale preference
 */
export function formatCurrencyLocalized(
  amount: number,
  currency?: string
): string {
  const locale = getUserLocale()
  const fallbackCurrency = currency || LOCALE_CURRENCY_MAP[locale] || 'USD'
  
  return formatCurrencyAutoPrecision(amount, fallbackCurrency, locale)
}