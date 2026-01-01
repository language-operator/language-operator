import { withAuth } from 'next-auth/middleware'

export default withAuth(
  function middleware(req) {
    // This function only runs for authenticated requests
    // For public routes, NextAuth won't even call this function
  },
  {
    callbacks: {
      authorized: ({ token, req }) => {
        const { pathname } = req.nextUrl
        
        // Allow public access to invitation pages
        if (pathname.startsWith('/invites/')) {
          return true
        }
        
        // Allow public access to invitation API endpoints
        if (pathname.startsWith('/api/invites/')) {
          return true
        }
        
        // Allow public access to auth pages
        if (pathname.startsWith('/login') || pathname.startsWith('/signup')) {
          return true
        }
        
        // Require authentication for all other routes
        return !!token
      },
    },
    pages: {
      signIn: '/login',
    },
  }
)