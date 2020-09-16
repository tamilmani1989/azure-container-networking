// -----------------------------------------------------------------------------
// <copyright file="AppInfo.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System;
    using System.IO;
    using System.Reflection;

    public static class AppInfo
    {
        private static string folderPath;
        private static string filePath;

        /// <summary>
        /// Folder path for the executable.
        /// </summary>

        public static string FolderPath
        {
            get
            {
                if (string.IsNullOrEmpty(folderPath))
                {
                    folderPath = Path.GetDirectoryName(FilePath);
                }

                return folderPath;
            }
        }

        /// <summary>
        /// Path to the executable.
        /// </summary>
        public static string FilePath
        {
            get
            {
                if (string.IsNullOrEmpty(filePath))
                {
                    Assembly assembly = Assembly.GetEntryAssembly();
                    if (assembly == null)
                    {
                        assembly = Assembly.GetExecutingAssembly();
                    }

                    filePath = assembly.Location;
                }

                return filePath;
            }
        }

        /// <summary>
        /// Search for file in known app base directories and return
        /// full path to the directory where file resides
        /// </summary>
        /// <param name="fileName"></param>
        /// <returns></returns>
        public static string GetFilePath(string fileName)
        {
            string[] dirs = { FolderPath, Directory.GetCurrentDirectory() };

            foreach (string d in dirs)
            {
                string filePath = Path.Combine(d, fileName);
                if (File.Exists(filePath))
                {
                    return d;
                }
            }

            throw new Exception($"File {fileName} not found in paths: {FolderPath}, {Directory.GetCurrentDirectory()}");
        }
    }
}
