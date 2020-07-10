/**
 * @file BondSerializer.h
 * @brief Implementation for BondSerializer.
 *
 * @copyright Copyright (c) Microsoft Corporation
 *
**/

#pragma once

#include <bond/core/bond.h> 
#include <bond/stream/output_buffer.h> 
#include <bond/stream/stdio_output_stream.h> 
#include <bond/protocol/simple_json_reader.h>
#include <bond/protocol/simple_json_writer.h>
#include <boost/algorithm/string.hpp>

#define BOND_FORMAT_COMPACT_BINARY L"CompactBinary"
#define BOND_FORMAT_JSON L"JSON"

class BondSerializer
{
public:
    template <typename T>
    static
    bool
    TrySerializeBuffer(bond::OutputBuffer &buffer, T &obj, std::wstring format)
    {
        try
        {
            SerializeBuffer(buffer, obj, format);
        }
        catch (...)
        {
            return false;
        }

        return true;
    }

    template <typename T>
    static
    bool
    TryDeSerializeBuffer(bond::InputBuffer &buffer, T &obj, std::wstring format)
    {
        try
        {
            DeSerializeBuffer(buffer, obj, format);
        }
        catch (...)
        {
            return false;
        }

        return true;
    }

private:
    template <typename T>
    static
    void
    SerializeBuffer(bond::OutputBuffer &buffer, T &obj, std::wstring format)
    {
        try
        {
            if (boost::iequals(format, BOND_FORMAT_COMPACT_BINARY))
            {
                bond::CompactBinaryWriter<bond::OutputBuffer> writer(buffer);
                bond::Serialize(obj, writer);
            } else if (boost::iequals(format, BOND_FORMAT_JSON))
            {
                bond::SimpleJsonWriter<bond::OutputBuffer> writer(buffer);
                bond::Serialize(obj, writer);
            }
            else
            {
                bond::CompactBinaryWriter<bond::OutputBuffer> writer(buffer);
                bond::Serialize(obj, writer);
            }
        }
        catch (...)
        {
            throw;
        }
    }

    template <typename T>
    static
    void
    DeSerializeBuffer(
        bond::InputBuffer &buffer, 
        T &obj, 
        std::wstring format
        )
    {
        try
        {
            if (boost::iequals(format, BOND_FORMAT_COMPACT_BINARY))
            {
                bond::CompactBinaryReader<bond::InputBuffer> reader(buffer);
                bond::Deserialize(reader, obj);
            } else if (boost::iequals(format, BOND_FORMAT_JSON))
            {
                bond::SimpleJsonReader<bond::InputBuffer> reader(buffer);
                bond::Deserialize(reader, obj);
            }
            else
            {
                bond::CompactBinaryReader<bond::InputBuffer> reader(buffer);
                bond::Deserialize(reader, obj);
            }
        }
        catch (...)
        {
            throw;
        }
    }
};
