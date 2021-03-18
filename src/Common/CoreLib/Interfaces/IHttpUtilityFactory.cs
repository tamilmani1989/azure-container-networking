

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
	using System;
	using System.Collections.Generic;
	using System.Text;

    public interface IHttpUtilityFactory
    {
        public void Register(string name, IHttpUtility utility);
        public IHttpUtility Resolve(string utility);

    }
}
