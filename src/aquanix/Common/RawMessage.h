#pragma once
#include "aquabase.h"

namespace aqua
{
	class RawMessage
	{
	private:
		UCHAR *m_data;
		USHORT m_size;

	public:
		RawMessage();
		RawMessage(UCHAR *data, USHORT messageSize, bool copyBuffer);
		~RawMessage();

		const char *GetData();
		USHORT GetDataSize();
	};
}