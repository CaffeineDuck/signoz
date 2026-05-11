import { ApiV2Instance as axios } from 'api';
import { ErrorResponseHandlerV2 } from 'api/ErrorResponseHandlerV2';
import { AxiosError } from 'axios';
import { ErrorV2Resp, RawSuccessResponse, SuccessResponseV2 } from 'types/api';
import { Token } from 'types/api/v2/sessions/email_password/post';

// Trades a trusted-header-authenticated request (e.g. with X-Authentik-Email
// injected by the reverse proxy) for a tokenizer-issued JWT in the same shape
// as the email/password flow returns. Body is empty — auth is in the headers.
const post = async (): Promise<SuccessResponseV2<Token>> => {
	try {
		const response = await axios.post<RawSuccessResponse<Token>>(
			'/sessions/trustedheader',
		);

		return {
			httpStatusCode: response.status,
			data: response.data.data,
		};
	} catch (error) {
		ErrorResponseHandlerV2(error as AxiosError<ErrorV2Resp>);
	}
};

export default post;
