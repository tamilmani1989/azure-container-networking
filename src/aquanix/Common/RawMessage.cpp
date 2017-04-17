#include "RawMessage.h"


RawMessage::RawMessage(UCHAR *data, USHORT messageSize, bool copyBuffer)
{
	m_size = messageSize;
	if (copyBuffer)
	{
		m_data = new UCHAR[m_size];
		memcpy(m_data, data, m_size);
	}
	else
	{
		m_data = data;
	}
}

RawMessage::~RawMessage()
{
	if (m_data != NULL)
	{
		delete[] m_data;
		m_data = NULL;
	}
	m_size = 0;
}

const char *RawMessage::GetData()
{
	return (const char *)m_data;
}

USHORT RawMessage::GetDataSize()
{
	return m_size;
}