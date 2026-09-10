export interface Review {
  id: string
  productId: string
  author: string
  rating: 1 | 2 | 3 | 4 | 5
  title: string
  body: string
  verifiedPurchase: boolean
  photos?: string[]
  helpfulCount: number
  notHelpfulCount: number
  createdAt: string
}

export interface RatingDistribution {
  5: number
  4: number
  3: number
  2: number
  1: number
}

export interface Question {
  id: string
  productId: string
  author: string
  question: string
  answer?: string
  answeredBy?: string
  createdAt: string
}
