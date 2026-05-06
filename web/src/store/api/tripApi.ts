import { createApi } from '@reduxjs/toolkit/query/react';
import type { BaseQueryFn } from '@reduxjs/toolkit/query';
import type { AxiosError, AxiosRequestConfig } from 'axios';
import apiClient from '../../lib/axios';
import {
  BackendEndpoints,
  HTTPTripPreviewRequestPayload,
  HTTPTripPreviewResponse,
  HTTPTripStartRequestPayload,
} from '../../contracts';
import { HTTPTripStartResponse } from '../../types';

interface AxiosBaseQueryArgs {
  url: string;
  method?: AxiosRequestConfig['method'];
  data?: AxiosRequestConfig['data'];
  params?: AxiosRequestConfig['params'];
}

const axiosBaseQuery: BaseQueryFn<AxiosBaseQueryArgs, unknown, unknown> = async ({
  url,
  method = 'GET',
  data,
  params,
}) => {
  try {
    const result = await apiClient({ url, method, data, params });
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

export const tripApi = createApi({
  reducerPath: 'tripApi',
  baseQuery: axiosBaseQuery,
  endpoints: (builder) => ({
    previewTrip: builder.mutation<{ data: HTTPTripPreviewResponse }, HTTPTripPreviewRequestPayload>({
      query: (payload) => ({
        url: BackendEndpoints.PREVIEW_TRIP,
        method: 'POST',
        data: payload,
      }),
    }),
    startTrip: builder.mutation<HTTPTripStartResponse, HTTPTripStartRequestPayload>({
      query: (payload) => ({
        url: BackendEndpoints.START_TRIP,
        method: 'POST',
        data: payload,
      }),
    }),
  }),
});

export const { usePreviewTripMutation, useStartTripMutation } = tripApi;
