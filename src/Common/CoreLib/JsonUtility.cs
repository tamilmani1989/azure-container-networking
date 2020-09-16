// -----------------------------------------------------------------------------
// <copyright file="JsonUtility.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System.Collections.Generic;
    using System.IO;
    using Newtonsoft.Json;
    using Newtonsoft.Json.Converters;

    public static class JsonUtility
    {
        /// <summary>
        /// The serialization settings.
        /// </summary>
        private static readonly JsonSerializerSettings serializationSettings;
        private static readonly JsonSerializerSettings serializationsettingsTypeHandling;
        private static readonly JsonSerializerSettings deserializationsettingsTypeHandling;

        static JsonUtility()
        {
            serializationSettings = new JsonSerializerSettings
            {
                Converters = new List<JsonConverter>
                {
                    new StringEnumConverter()
                }
            };

            serializationsettingsTypeHandling = new JsonSerializerSettings
            {
                TypeNameHandling = TypeNameHandling.All,
                Converters = new List<JsonConverter>
                {
                    new StringEnumConverter()
                }
            };

            deserializationsettingsTypeHandling = new JsonSerializerSettings
            {
                TypeNameHandling = TypeNameHandling.All,
            };
        }

        /// <summary>
        /// Deserialzation is done with default settings. It means TypeHandling is not
        /// used and json won't get deserialized  properly into a derived class. For derived class
        /// Serialization/Deserialization use API DeserializeWithTypeHandling and SerializeWithTypeHandling
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="json"></param>
        /// <returns></returns>
        public static T Deserialize<T>(string json)
        {
            return JsonConvert.DeserializeObject<T>(json);
        }

        /// <summary>
        /// Handles deserialation into derived classes. For this to work,
        /// make sure serialzation was also done using type handling (SerializeWithTypeHandling method in this class)
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="json"></param>
        /// <returns></returns>
        public static T DeserializeWithTypeHandling<T>(string json)
        {
            return JsonConvert.DeserializeObject<T>(json, deserializationsettingsTypeHandling);
        }

        public static T Deserialize<T>(Stream s)
        {
            using (StreamReader reader = new StreamReader(s))
            using (JsonTextReader jsonReader = new JsonTextReader(reader))
            {
                JsonSerializer ser = new JsonSerializer();
                return ser.Deserialize<T>(jsonReader);
            }
        }

        public static string Serialize(object input)
        {
            return JsonConvert.SerializeObject(input, serializationSettings);
        }

        public static string SerializeWithTypeHandling(object input)
        {
            return JsonConvert.SerializeObject(input, serializationsettingsTypeHandling);
        }
    }
}
