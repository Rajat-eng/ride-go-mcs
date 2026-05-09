import { createApi } from '@reduxjs/toolkit/query/react';
import type { BaseQueryFn } from '@reduxjs/toolkit/query';
import type { AxiosError, AxiosRequestConfig } from 'axios';
import apiClient from '../../lib/axios';
import { AuthUser } from '../slices/authSlice';

interface AxiosBaseQueryArgs {
  url: string;
  method?: AxiosRequestConfig['method'];
  data?: AxiosRequestConfig['data'];
}

const axiosBaseQuery: BaseQueryFn<AxiosBaseQueryArgs, unknown, unknown> = async ({
  url,
  method = 'GET',
  data,
}) => {
  try {
    const result = await apiClient({ url, method, data });
    return { data: result.data };
  } catch (axiosError) {
    const err = axiosError as AxiosError;
    return {
      error: {
        status: err.response?.status,
        data: err.response?.data || err.message,
      },
    };
  }
};

export interface SignupPayload {
  email: string;
  password: string;
  name: string;
  phoneNumber?: string;
}

export interface LoginPayload {
  email: string;
  password: string;
}

export interface AuthResponse {
  data: {
    accessToken: string;
    user: AuthUser;
  };
}

export const authApi = createApi({
  reducerPath: 'authApi',
  baseQuery: axiosBaseQuery,
  endpoints: (builder) => ({
    signup: builder.mutation<AuthResponse, SignupPayload>({
      query: (payload) => ({
        url: '/auth/signup',
        method: 'POST',
        data: payload,
      }),
    }),
    login: builder.mutation<AuthResponse, LoginPayload>({
      query: (payload) => ({
        url: '/auth/login',
        method: 'POST',
        data: payload,
      }),
    }),
    refreshToken: builder.mutation<{ data: { accessToken: string; refreshToken: string } }, { refreshToken: string }>({
      query: (payload) => ({
        url: '/auth/refresh',
        method: 'POST',
        data: payload,
      }),
    }),
  }),
});

export const { useSignupMutation, useLoginMutation, useRefreshTokenMutation } = authApi;
