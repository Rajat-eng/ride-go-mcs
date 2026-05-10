import type { AuthOptions } from 'next-auth';
import GoogleProvider from 'next-auth/providers/google';

export const authOptions: AuthOptions = {
  providers: [
    GoogleProvider({
      clientId: process.env.GOOGLE_CLIENT_ID ?? '',
      clientSecret: process.env.GOOGLE_CLIENT_SECRET ?? '',
    }),
  ],
  secret: process.env.NEXTAUTH_SECRET,
  session: {
    strategy: 'jwt',
  },
  callbacks: {
    async jwt({ token, account }) {
      // Include the Google ID token in the JWT so it can be accessed in the session callback for inclusion in the session and sent to the client. 
      // This is needed because next-auth does not persist the account object after the initial sign in, but we need the ID token from the Google provider to exchange for our own access token on the backend.
      // when clicking Google provider, next-auth will handle the OAuth flow and get the ID token from Google, then pass it to this callback in the account object
      if (account?.provider === 'google' && account.id_token) {
        token.googleIdToken = account.id_token;
      }
      return token;
    },
    async session({ session, token }) {
      // Include the Google ID token in the session so it can be accessed on the client side
      if (typeof token.googleIdToken === 'string') {
        session.googleIdToken = token.googleIdToken;
      }
      return session; // now call backend api to exchange Google ID token for our own access token and create a session for the user
    },
  },
};
